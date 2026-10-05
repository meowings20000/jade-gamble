package store

// AIPoolStore: Claude 共產池（喵喵幣捐贈 → 動態門檻 → 解鎖/上鎖）
//
// 玩家捐喵喵幣進公共池 → 達到階梯門檻 → DMIT 端開啟 claude 頻道
// 額度剩越少 → 門檻越高（動態）
// 7 天剩餘 < 底線 → 無條件鎖死
// DMIT cron 每 5 分鐘打 /api/ai/pool/sync 回報用量、拿指令。

import (
	"database/sql"
	"errors"
	"time"
)

// AIUsageSnapshot 一份 Anthropic 官方窗口用量（DMIT cron 推上來）
type AIUsageSnapshot struct {
	FiveHourUtil  float64 `json:"five_hour_util"`  // 5h 窗口已用 %
	FiveHourReset string  `json:"five_hour_reset"` // ISO8601
	SevenDayUtil  float64 `json:"seven_day_util"`  // 7d 窗口已用 %
	SevenDayReset string  `json:"seven_day_reset"` // ISO8601
}

// AIPoolView 給前端 / DMIT 看的池子狀態
type AIPoolView struct {
	PoolChips     int64   `json:"pool_chips"`     // 池子現況
	Threshold     int64   `json:"threshold"`      // 目前動態門檻
	Unlocked      bool    `json:"unlocked"`       // pool >= threshold
	MyContrib     int64   `json:"my_contrib"`     // 我捐了多少
	SevenDayUtil  float64 `json:"seven_day_util"` // 7 天窗口已用 %
	FiveHourUtil  float64 `json:"five_hour_util"` // 5 小時窗口已用 %
	SevenDayReset string  `json:"seven_day_reset"`
	FiveHourReset string  `json:"five_hour_reset"`
	FloorLock     bool    `json:"floor_lock"` // 底線鎖死
	// ChipsAfter 只在 contribute 回應時填（客端用來更新餘額顯示）
	ChipsAfter int64 `json:"chips_after,omitempty"`
}

// --- 門檻階梯（跟 7d 剩餘 % 掛勾）---
// 剩餘 >=80%  → 500,000（起跳價）
// 剩餘 60-80% → 750,000
// 剩餘 40-60% → 1,000,000
// 剩餘 25-40% → 1,500,000
// 剩餘 15-25% → 2,000,000
// 剩餘 <15%   → 底線鎖死（門檻視為無限大）
const AIFloorLockPct = 15.0

// 單人捐贈計入池子的上限（佔門檻的百分比）：防止一人獨扛全服
// 實際計入 = min(捐額, threshold * 20%)
const AIContribCapPct = 20.0

func aiThreshold(sevenDayUtil float64) int64 {
	remain := 100.0 - sevenDayUtil
	switch {
	case remain < AIFloorLockPct:
		return 1 << 62
	case remain < 25:
		return 2_000_000
	case remain < 40:
		return 1_500_000
	case remain < 60:
		return 1_000_000
	case remain < 80:
		return 750_000
	default:
		return 500_000
	}
}

// AIPoolEnsure 建表
func (s *Store) AIPoolEnsure() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ai_pool (
		id INTEGER PRIMARY KEY CHECK (id=1),
		pool_chips INTEGER NOT NULL DEFAULT 0,
		seven_day_util REAL NOT NULL DEFAULT 0,
		five_hour_util REAL NOT NULL DEFAULT 0,
		seven_day_reset TEXT NOT NULL DEFAULT '',
		five_hour_reset TEXT NOT NULL DEFAULT '',
		unlocked INTEGER NOT NULL DEFAULT 0,
		unlocked_at TEXT NOT NULL DEFAULT '',
		last_sync TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ai_contrib (
		user_id INTEGER PRIMARY KEY,
		total INTEGER NOT NULL DEFAULT 0,
		last_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return err
	}
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ai_pool_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		amount INTEGER NOT NULL,
		pool_after INTEGER NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	return err
}

// aiEffectivePool 計入單人上限後的「有效池子」：
// 每人計入額 = min(該人累計捐贈, threshold 的 20%)，總和即有效池
// （防止一人獨扛全服 —— 想開 Claude 至少要 5 個人出力）
func (s *Store) aiEffectivePool(threshold int64) (int64, error) {
	cap := threshold * 20 / 100
	rows, err := s.db.Query(`SELECT total FROM ai_contrib`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var eff int64
	for rows.Next() {
		var t int64
		if err := rows.Scan(&t); err != nil {
			return 0, err
		}
		if t > capWithCap(t, cap) {
			eff += capWithCap(t, cap)
		} else {
			eff += t
		}
	}
	return eff, rows.Err()
}

// capWithCap 單人計入上限
func capWithCap(t, cap int64) int64 {
	if t > cap {
		return cap
	}
	return t
}

// AIPoolGet 讀池子狀態
func (s *Store) AIPoolGet() (pool int64, u7, u5 float64, r7, r5 string, unlocked bool, err error) {
	row := s.db.QueryRow(`SELECT pool_chips, seven_day_util, five_hour_util, seven_day_reset, five_hour_reset, unlocked FROM ai_pool WHERE id=1`)
	err = row.Scan(&pool, &u7, &u5, &r7, &r5, &unlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, "", "", false, nil
	}
	return
}

// AIContributeTx 捐喵喵幣進池子（tx 內扣 chips、加總、寫 log）
// 回傳 (pool_after, my_chips_after)
func (s *Store) AIContributeTx(tx *Tx, userID int, amount int64) (int64, int64, error) {
	if amount <= 0 {
		return 0, 0, ErrInsufficient
	}
	var chips int64
	if err := tx.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&chips); err != nil {
		return 0, 0, err
	}
	if chips < amount {
		return 0, 0, ErrInsufficient
	}
	newChips := chips - amount
	if _, err := tx.Exec(`UPDATE users SET chips=? WHERE id=?`, newChips, userID); err != nil {
		return 0, 0, err
	}
	var pool int64
	err := tx.QueryRow(`SELECT pool_chips FROM ai_pool WHERE id=1`).Scan(&pool)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.Exec(`INSERT INTO ai_pool (id, pool_chips) VALUES (1, ?)`, amount); err != nil {
			return 0, 0, err
		}
		pool = amount
	} else if err != nil {
		return 0, 0, err
	} else {
		if _, err := tx.Exec(`UPDATE ai_pool SET pool_chips=pool_chips+? WHERE id=1`, amount); err != nil {
			return 0, 0, err
		}
		pool += amount
	}
	if _, err := tx.Exec(`INSERT INTO ai_contrib (user_id, total, last_at) VALUES (?,?,datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET total=total+excluded.total, last_at=excluded.last_at`, userID, amount); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(`INSERT INTO ai_pool_log (user_id, amount, pool_after) VALUES (?,?,?)`, userID, amount, pool); err != nil {
		return 0, 0, err
	}
	return pool, newChips, nil
}

// AISync DMIT cron 回報用量；遊戲端決定 unlock/lock 指令。
// unlock: 池子 >= 門檻 且未解鎖 且沒踩底線
// lock:   已解鎖但 (池子 < 門檻 或踩底線)
func (s *Store) AISync(snap AIUsageSnapshot, now time.Time) (view AIPoolView, unlock, lock bool, err error) {
	if e := s.AIPoolEnsure(); e != nil {
		return view, false, false, e
	}
	if _, e := s.db.Exec(`INSERT INTO ai_pool (id, seven_day_util, five_hour_util, seven_day_reset, five_hour_reset, last_sync)
		VALUES (1,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET seven_day_util=excluded.seven_day_util,
			five_hour_util=excluded.five_hour_util,
			seven_day_reset=excluded.seven_day_reset,
			five_hour_reset=excluded.five_hour_reset,
			last_sync=excluded.last_sync`,
		snap.SevenDayUtil, snap.FiveHourUtil, snap.SevenDayReset, snap.FiveHourReset, now.UTC().Format(time.RFC3339)); e != nil {
		return view, false, false, e
	}
	_, u7, u5, r7, r5, unlocked, e := s.AIPoolGet()
	if e != nil {
		return view, false, false, e
	}
	th := aiThreshold(u7)
	floor := (100.0 - u7) < AIFloorLockPct
	// 5h 窗口撞牆（>=90%）→ 暫時鎖（等窗口重置），錢不退
	// 這避免「空開門」：池子達標但開出去一分鐘就限速
	fiveHourWall := u5 >= 90.0
	// 有效池（單人封頂 20% 後）才是解鎖依據
	effPool, e := s.aiEffectivePool(th)
	if e != nil {
		return view, false, false, e
	}
	switch {
	case !unlocked && effPool >= th && !floor && !fiveHourWall:
		if _, e := s.db.Exec(`UPDATE ai_pool SET unlocked=1, unlocked_at=? WHERE id=1`, now.UTC().Format(time.RFC3339)); e != nil {
			return view, false, false, e
		}
		unlocked = true
		unlock = true
	case unlocked && (effPool < th || floor || fiveHourWall):
		if _, e := s.db.Exec(`UPDATE ai_pool SET unlocked=0 WHERE id=1`); e != nil {
			return view, false, false, e
		}
		unlocked = false
		lock = true
	}
	return AIPoolView{
		PoolChips: effPool, Threshold: th, Unlocked: unlocked,
		SevenDayUtil: u7, FiveHourUtil: u5,
		SevenDayReset: r7, FiveHourReset: r5, FloorLock: floor,
	}, unlock, lock, nil
}

// AIView 給前端 / DMIT 查池子（userID 0 = 匿名，不顯示 my_contrib）
func (s *Store) AIView(userID int) (AIPoolView, error) {
	_, u7, u5, r7, r5, unlocked, err := s.AIPoolGet()
	if err != nil {
		return AIPoolView{}, err
	}
	var my int64
	if userID > 0 {
		_ = s.db.QueryRow(`SELECT total FROM ai_contrib WHERE user_id=?`, userID).Scan(&my)
	}
	th := aiThreshold(u7)
	floor := (100.0 - u7) < AIFloorLockPct
	effPool, e := s.aiEffectivePool(th)
	if e != nil {
		return AIPoolView{}, e
	}
	return AIPoolView{
		PoolChips: effPool, Threshold: th, Unlocked: unlocked, MyContrib: my,
		SevenDayUtil: u7, FiveHourUtil: u5,
		SevenDayReset: r7, FiveHourReset: r5, FloorLock: floor,
	}, nil
}

// AILockForce 管理員 / 守門員強制上鎖
func (s *Store) AILockForce() error {
	_, err := s.db.Exec(`UPDATE ai_pool SET unlocked=0 WHERE id=1`)
	return err
}
