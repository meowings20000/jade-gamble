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
	"fmt"
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
	// 開市計時：非空且在未來 = 開市中；過期 = 收市
	OpenUntil string `json:"open_until"`
	// ChipsAfter 只在 contribute 回應時填（客端用來更新餘額顯示）
	ChipsAfter int64 `json:"chips_after,omitempty"`
}

// --- 門檻階梯（跟 7d 剩餘 % 掛勾）---
// 2026-10-06 測試期間：固定 500,000（最快能測到按鈕）
// （正式階梯之後再調回；AIFloorLockPct = 15% 底線鎖死）
const AIFloorLockPct = 15.0

// 單人捐贈上限（浮動）：每人累計 ≤ 全池總捐的 20%（別人捐多你也能捐多）
func aiThreshold(sevenDayUtil float64) int64 {
	remain := 100.0 - sevenDayUtil
	if remain < AIFloorLockPct {
		return 1 << 62 // 底線鎖死：實際上達不到
	}
	return 500_000 // 測試期固定門檻
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
		open_until TEXT NOT NULL DEFAULT '',
		last_sync TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return err
	}
	// 舊庫補欄位（2026-10-06 開市計時）
	if _, err := s.db.Exec(`ALTER TABLE ai_pool ADD COLUMN open_until TEXT NOT NULL DEFAULT ''`); err != nil {
		// 已存在會報 duplicate column —— 忽略
		var _ = err
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

// aiEffectivePool 計入單人浮動上限後的「有效池子」：
// ★ 規則（user 2026-10-06 定）：每人累計 ≤ 全池總捐的 20%（浮動）
//
//	別人捐越多、每人的容許額越大；防止一人獨扛全服。
//	解鎖判定用「有效池」：單人超額的部分不計入（錢已收，只是判定時不算）。
//	pool_chips 永遠 = 真實收到總額；解鎖看 effPool。
func (s *Store) aiEffectivePool(threshold int64) (int64, error) {
	total, err := s.aiTotalContrib()
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	cap := total * 20 / 100 // 浮動：全池的 20%
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
		eff += capWithCap(t, cap)
	}
	return eff, rows.Err()
}

// aiTotalContrib 全池真實總捐（所有 ai_contrib 加總）
func (s *Store) aiTotalContrib() (int64, error) {
	var t int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM ai_contrib`).Scan(&t)
	return t, err
}

// capWithCap 單人計入上限
func capWithCap(t, cap int64) int64 {
	if t > cap {
		return cap
	}
	return t
}

// AIPoolGet 讀池子狀態
func (s *Store) AIPoolGet() (pool int64, u7, u5 float64, r7, r5 string, unlocked bool, openUntil string, err error) {
	row := s.db.QueryRow(`SELECT pool_chips, seven_day_util, five_hour_util, seven_day_reset, five_hour_reset, unlocked, IFNULL(open_until,'') FROM ai_pool WHERE id=1`)
	err = row.Scan(&pool, &u7, &u5, &r7, &r5, &unlocked, &openUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, "", "", false, "", nil
	}
	return
}

// AIMarketOpen 玩家按鈕：池子達標 → 開市一小時、池子清零
// 回傳 (newView, error)。重複開市/未達標都會被擋。
func (s *Store) AIMarketOpen(userID int, now time.Time) (AIPoolView, error) {
	if e := s.AIPoolEnsure(); e != nil {
		return AIPoolView{}, e
	}
	_, u7, u5, _, _, unlocked, _, err := s.AIPoolGet()
	if err != nil {
		return AIPoolView{}, err
	}
	th := aiThreshold(u7)
	floor := (100.0 - u7) < AIFloorLockPct
	fiveHourWall := u5 >= 90.0
	effPool, e := s.aiEffectivePool(th)
	if e != nil {
		return AIPoolView{}, e
	}
	if floor || fiveHourWall {
		return AIPoolView{}, errors.New("額度底線 / 5小時撞牆，不能開市")
	}
	if unlocked {
		return AIPoolView{}, errors.New("已經開市中")
	}
	if effPool < th {
		return AIPoolView{}, fmt.Errorf("池子未達門檻（%d / %d）", effPool, th)
	}
	// 開市：開一小時、池子清零（捐贈＝買下這一小時）
	openUntil := now.UTC().Add(1 * time.Hour).Format(time.RFC3339)
	if _, e := s.db.Exec(`UPDATE ai_pool SET unlocked=1, unlocked_at=?, open_until=?, pool_chips=0 WHERE id=1`,
		now.UTC().Format(time.RFC3339), openUntil); e != nil {
		return AIPoolView{}, e
	}
	v, e := s.AIView(userID)
	if e != nil {
		return AIPoolView{}, e
	}
	v.Unlocked = true
	v.OpenUntil = openUntil
	return v, nil
}

// AIContributeTx 捐喵喵幣進池子（tx 內扣 chips、加總、寫 log）
// ★ 單人硬上限：我的「計入額」= min(累計捐, cap)；超過 cap 的部分「捐了也只算 cap」
//
//	且達 cap 後直接拒絕（user 2026-10-06：「捐超過20%仍然還能再捐」是 bug）
//
// 回傳 (pool_after, my_chips_after)
func (s *Store) AIContributeTx(tx *Tx, userID int, amount int64) (int64, int64, error) {
	if amount <= 0 {
		return 0, 0, ErrInsufficient
	}
	// ★ 浮動 cap（user 2026-10-06）：我的累計 + 這次 ≤ 全池總捐 × 20%
	//   全池 = 已收到的真實總捐（ai_contrib SUM）；我還沒捐過 → 我能捐到「目前全池×20%」
	//   註：全池 < 我的累計×5 時會擋 → 推著其他人也捐（浮動的意義）
	var total int64
	_ = tx.QueryRow(`SELECT COALESCE(SUM(total),0) FROM ai_contrib`).Scan(&total)
	var myAlready int64
	_ = tx.QueryRow(`SELECT total FROM ai_contrib WHERE user_id=?`, userID).Scan(&myAlready)
	// 我的計入基準 = min(我的累計, 目前 cap)（超歷史額時以 cap 為準）
	cap := total * 20 / 100
	if myAlready >= cap {
		return 0, 0, fmt.Errorf("單人上限：池子 20%（目前 %s）—— 邀更多人捐，你的容許額會變大", shortNum(cap))
	}
	if myAlready+amount > cap {
		return 0, 0, fmt.Errorf("超出單人上限：這次最多還能捐 %s（池子越大你能捐越多）", shortNum(cap-myAlready))
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
	_, u7, u5, r7, r5, unlocked, openUntil, e := s.AIPoolGet()
	if e != nil {
		return view, false, false, e
	}
	th := aiThreshold(u7)
	floor := (100.0 - u7) < AIFloorLockPct
	// 5h 撞牆（>=90%）不再收市（開市中不能關）—— 只留給前端顯示狀態
	_ = u5
	// 開市計時：open_until 未到 = 開市中（即使池子被清零也維持 unlock）
	openUntilTime := parseRFC3339(openUntil)
	marketOpen := unlocked && openUntilTime.After(now)
	// 有效池（單人封頂後）
	effPool, e := s.aiEffectivePool(th)
	if e != nil {
		return view, false, false, e
	}
	switch {
	case marketOpen:
		// ★ 開市中（open_until 未過）→ 一律維持，不能關（user 2026-10-06:「開著的情況 不能關掉」)
		// 保險絲（底線/5h 撞牆）也不收市 —— 撞牆時反正打不出去，等窗口重置即可
	case unlocked && !marketOpen:
		// open_until 過期 → 收市
		if _, e := s.db.Exec(`UPDATE ai_pool SET unlocked=0, open_until='' WHERE id=1`); e != nil {
			return view, false, false, e
		}
		unlocked = false
		lock = true
	case !unlocked && effPool >= th:
		// 池子達標（按鈕待按）—— 不自動開，等玩家按鈕
	}
	return AIPoolView{
		PoolChips: effPool, Threshold: th, Unlocked: unlocked,
		SevenDayUtil: u7, FiveHourUtil: u5,
		SevenDayReset: r7, FiveHourReset: r5, FloorLock: floor,
		OpenUntil: openUntil,
	}, unlock, lock, nil
}

// AIView 給前端 / DMIT 查池子（userID 0 = 匿名，不顯示 my_contrib）
// PoolChips = ai_pool 真實餘額（開市扣款後=0）—— 前端主顯示
func (s *Store) AIView(userID int) (AIPoolView, error) {
	rawPool, u7, u5, r7, r5, unlocked, openUntilDB, err := s.AIPoolGet()
	if err != nil {
		return AIPoolView{}, err
	}
	var my int64
	if userID > 0 {
		_ = s.db.QueryRow(`SELECT total FROM ai_contrib WHERE user_id=?`, userID).Scan(&my)
	}
	th := aiThreshold(u7)
	floor := (100.0 - u7) < AIFloorLockPct
	// AIView 回真實餘額（rawPool）；開市中 rawPool=0 正確呈現「已燒掉」
	_ = th
	_ = floor
	return AIPoolView{
		PoolChips: rawPool, Threshold: th, Unlocked: unlocked, MyContrib: my,
		SevenDayUtil: u7, FiveHourUtil: u5,
		SevenDayReset: r7, FiveHourReset: r5, FloorLock: floor,
		OpenUntil: openUntilDB,
	}, nil
}

// AILockForce 管理員 / 守門員強制上鎖
func (s *Store) AILockForce() error {
	_, err := s.db.Exec(`UPDATE ai_pool SET unlocked=0, open_until='' WHERE id=1`)
	return err
}

func parseRFC3339(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func shortNum(n int64) string {
	switch {
	case n >= 100_000_000:
		return fmt.Sprintf("%.1f億", float64(n)/100_000_000)
	case n >= 10_000:
		return fmt.Sprintf("%d萬", n/10_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// AIContribList 捐獻名單（榜）：username、總捐、佔比、最後時間
func (s *Store) AIContribList(limit int) ([]map[string]any, error) {
	total, err := s.aiTotalContrib()
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT a.user_id, u.username, a.total, a.last_at
		FROM ai_contrib a LEFT JOIN users u ON u.id=a.user_id
		ORDER BY a.total DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var uid int64
		var name string
		var lastAt string
		var amt int64
		if err := rows.Scan(&uid, &name, &amt, &lastAt); err != nil {
			return nil, err
		}
		pct := 0.0
		if total > 0 {
			pct = float64(amt) / float64(total) * 100
		}
		if name == "" {
			name = "神秘玩家"
		}
		out = append(out, map[string]any{
			"user_id": uid, "username": name, "amount": amt,
			"pct": pct, "last_at": lastAt,
		})
	}
	return out, rows.Err()
}
