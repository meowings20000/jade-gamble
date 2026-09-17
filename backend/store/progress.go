package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"jade-gamble/backend/domain"
)

// Tx is the transaction handle passed to API fns (alias of sql.Tx).
type Tx = sql.Tx

// WithTx runs fn in a transaction.
func (s *Store) WithTx(fn func(tx *Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ---------- scratch progress ----------
//
// Layout: the crack layout is generated ONCE at scratch start with
// crypto-random and persisted — never derived from the public seed
// (which would leak crack positions to the client).

type ScratchProgress struct {
	StoneID     string
	UserID      int
	Revealed    map[int]bool
	Order       []int // 開格順序（累積值與順序有關）
	Accumulated int
	Done        bool
	Layout      ScratchLayout // persisted truth; empty for legacy rows
}

// ScratchLayout is the persisted server-side crack placement.
type ScratchLayout struct {
	CrackAt map[int]bool `json:"crack_at"`
	DeepAt  map[int]bool `json:"deep_at"`
}

func (l ScratchLayout) KindAt(cell int) string {
	switch {
	case l.DeepAt[cell]:
		return "deep_crack"
	case l.CrackAt[cell]:
		return "crack"
	default:
		return "clean"
	}
}

func (p *ScratchProgress) RevealedList() []int {
	out := []int{}
	for c := range p.Revealed {
		if p.Revealed[c] {
			out = append(out, c)
		}
	}
	return out
}

// RevealedKinds returns the kind of each already-revealed cell so the
// client can repaint a resumed session truthfully.
func (p *ScratchProgress) RevealedKinds() []map[string]any {
	out := []map[string]any{}
	for c := range p.Revealed {
		if p.Revealed[c] {
			out = append(out, map[string]any{"cell": c, "kind": p.Layout.KindAt(c)})
		}
	}
	return out
}

// SellNowFee applies the 4% early-stop fee to the stored accumulation.
func (p *ScratchProgress) SellNowFee() int {
	if p.Done {
		return p.Accumulated
	}
	return int(float64(p.Accumulated) * 0.96)
}

func (s *Store) GetScratchProgress(stoneID string) (*ScratchProgress, error) {
	row := s.db.QueryRow(`SELECT stone_id, user_id, revealed, accumulated, done, layout, order_cells
		FROM scratch_progress WHERE stone_id=?`, stoneID)
	var p ScratchProgress
	var rev, layout, order string
	var done int
	err := row.Scan(&p.StoneID, &p.UserID, &rev, &p.Accumulated, &done, &layout, &order)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Done = done == 1
	p.Revealed = map[int]bool{}
	_ = json.Unmarshal([]byte(rev), &p.Revealed)
	p.Order = []int{}
	_ = json.Unmarshal([]byte(order), &p.Order)
	if len(p.Order) == 0 {
		// 舊資料沒有順序：用格號排序當作當時的順序（近似還原）
		for c := 0; c < domain.ScratchCells; c++ {
			if p.Revealed[c] {
				p.Order = append(p.Order, c)
			}
		}
	}
	p.Layout = ScratchLayout{CrackAt: map[int]bool{}, DeepAt: map[int]bool{}}
	_ = json.Unmarshal([]byte(layout), &p.Layout)
	return &p, nil
}

func (s *Store) SaveScratchProgressFull(stoneID string, userID int, ss *domain.ScratchState, layout ScratchLayout) error {
	rev, _ := json.Marshal(ss.Revealed)
	lay, _ := json.Marshal(layout)
	order, _ := json.Marshal(ss.Order)
	_, err := s.db.Exec(`INSERT INTO scratch_progress (stone_id, user_id, revealed, accumulated, done, layout, order_cells)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(stone_id) DO UPDATE SET revealed=excluded.revealed, accumulated=excluded.accumulated,
			done=excluded.done, layout=excluded.layout, order_cells=excluded.order_cells`,
		stoneID, userID, string(rev), ss.Accumulated, b2i(ss.Done), string(lay), string(order))
	return err
}

// ---------- polish progress (stone) ----------

type PolishProgress struct {
	StoneID  string
	UserID   int
	Stage    int
	Alive    bool
	BreakMod float64
	Force    int // 力度 chosen before the wheel started; fixed for the run
}

func (s *Store) GetPolishProgress(stoneID string) (*PolishProgress, error) {
	row := s.db.QueryRow(`SELECT stone_id, user_id, stage, alive, break_mod, force FROM polish_progress WHERE stone_id=?`, stoneID)
	var p PolishProgress
	var alive int
	err := row.Scan(&p.StoneID, &p.UserID, &p.Stage, &alive, &p.BreakMod, &p.Force)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Alive = alive == 1
	if p.Force == 0 {
		p.Force = domain.PolishForceNormal
	}
	return &p, nil
}

func (s *Store) SavePolishProgress(stoneID string, userID, stage int, alive bool, breakMod float64, force int) error {
	_, err := s.db.Exec(`INSERT INTO polish_progress (stone_id, user_id, stage, alive, break_mod, force)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(stone_id) DO UPDATE SET stage=excluded.stage, alive=excluded.alive,
			break_mod=excluded.break_mod, force=excluded.force`,
		stoneID, userID, stage, b2i(alive), breakMod, force)
	return err
}

// PolishRunningCount: 這名玩家目前有幾顆石頭在磨（進行中、還沒落袋或磨崩）。
func (s *Store) PolishRunningCount(userID int) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM polish_progress WHERE user_id=? AND alive=1`, userID).Scan(&n)
	return n, err
}

// PolishRunningStone: 正在磨的那顆石頭 id（沒有就回空字串）。
func (s *Store) PolishRunningStone(userID int) (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT stone_id FROM polish_progress WHERE user_id=? AND alive=1 ORDER BY stone_id LIMIT 1`, userID).Scan(&id)
	if err != nil {
		return "", nil
	}
	return id, nil
}

// FinishPolish: 這一輪結束了（落袋或磨崩），把它從「進行中」拿掉。
// 少了這步，玩家結算後還是會被「一次只能磨一顆」擋住（2026-09-17 玩家 123aaa 回報卡住）。
func (s *Store) FinishPolish(stoneID string) error {
	_, err := s.db.Exec(`UPDATE polish_progress SET alive=0 WHERE stone_id=?`, stoneID)
	return err
}

// FinishPolishTx: 同上，但在既有交易內使用。
// 注意：連線池只有 1 條，交易裡面絕對不能再呼叫會自己拿連線的 Store 方法（會死鎖）。
func (s *Store) FinishPolishTx(tx *sql.Tx, stoneID string) error {
	_, err := tx.Exec(`UPDATE polish_progress SET alive=0 WHERE stone_id=?`, stoneID)
	return err
}

// PolishRunningStones: 真的還能續磨的進行中石頭（石頭還在倉庫、狀態 owned）。
// 2026-09-17 玩家 123aaa 卡死：石頭早就結算掉了，polish_progress 的幽靈紀錄
// 還掛著 alive=1，於是「一次只能磨一顆」把玩家永遠擋在外面。
func (s *Store) PolishRunningStones(userID int) ([]string, error) {
	rows, err := s.db.Query(`SELECT p.stone_id FROM polish_progress p
		JOIN stones s ON s.id = p.stone_id
		WHERE p.user_id = ? AND p.alive = 1 AND s.owner_id = ? AND s.state = 'owned'
		ORDER BY p.stone_id`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// PurgeStalePolish: 清掉幽靈紀錄（石頭已經不在倉庫或已處理完）。
func (s *Store) PurgeStalePolish(userID int) error {
	_, err := s.db.Exec(`DELETE FROM polish_progress
		WHERE user_id = ? AND alive = 1
		AND stone_id NOT IN (SELECT id FROM stones WHERE owner_id = ? AND state = 'owned')`, userID, userID)
	return err
}
