package store

import (
	"database/sql"

	"jade-gamble/backend/domain"
)

// 石頭處理紀錄（2026-09-16）——玩家可以看到自己切／刮／磨了哪些石頭、賺賠多少。
//
// 只記「這顆石頭被我處理掉」的結果，不記還沒處理的（那在倉庫看得到）。

// StoneLog: 一筆處理紀錄。
type StoneLog struct {
	ID        int    `json:"id"`
	StoneID   string `json:"stone_id"`
	Action    string `json:"action"` // cut / scratch / polish / sold
	Grade     int    `json:"grade"`
	Quality   string `json:"quality"`
	Variety   string `json:"variety"`
	Price     int    `json:"price"`
	Payout    int    `json:"payout"`
	CreatedAt string `json:"created_at"`
}

// HistoryStats: 這個人的總帳。
type HistoryStats struct {
	Total    int `json:"total"`
	Cuts     int `json:"cuts"`
	Wins     int `json:"wins"`
	Spent    int `json:"spent"`
	Earned   int `json:"earned"`
	Net      int `json:"net"`
	WinRate  int `json:"win_rate"`  // 百分比
	BestMult int `json:"best_mult"` // 最佳倍率 ×100
}

// LogStoneTx: 寫一筆處理紀錄。
func (s *Store) LogStoneTx(tx *sql.Tx, userID int, stoneID, action string, grade int, quality, variety string, price, payout int) error {
	_, err := tx.Exec(`INSERT INTO stone_log (user_id, stone_id, action, grade, quality, variety, price, payout)
		VALUES (?,?,?,?,?,?,?,?)`, userID, stoneID, action, grade, quality, variety, price, payout)
	return err
}

// ListHistory: 最近 N 筆處理紀錄。
func (s *Store) ListHistory(userID, limit int) ([]StoneLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	rows, err := s.db.Query(`SELECT id, stone_id, action, grade, quality, variety, price, payout, created_at
		FROM stone_log WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoneLog{}
	for rows.Next() {
		var l StoneLog
		if err := rows.Scan(&l.ID, &l.StoneID, &l.Action, &l.Grade, &l.Quality, &l.Variety,
			&l.Price, &l.Payout, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// HistoryStats: 統計（切石為主，含刮／磨／賣）。
func (s *Store) HistoryStats(userID int) (HistoryStats, error) {
	var st HistoryStats
	row := s.db.QueryRow(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN action='cut' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN payout >= price THEN 1 ELSE 0 END),0),
		COALESCE(SUM(price),0), COALESCE(SUM(payout),0)
		FROM stone_log WHERE user_id = ?`, userID)
	if err := row.Scan(&st.Total, &st.Cuts, &st.Wins, &st.Spent, &st.Earned); err != nil {
		return st, err
	}
	st.Net = st.Earned - st.Spent
	if st.Total > 0 {
		st.WinRate = st.Wins * 100 / st.Total
	}
	var best *float64
	_ = s.db.QueryRow(`SELECT MAX(CAST(payout AS REAL) / price) FROM stone_log WHERE user_id = ? AND price > 0`, userID).Scan(&best)
	if best != nil {
		st.BestMult = int(*best * 100)
	}
	return st, nil
}

// QualityName / VarietyName helpers so the API can stay thin.
func QualityName(q int) string { return domain.Quality(q).Name() }
func VarietyName(v int) string { return domain.ColorVariety(v).Name() }

// TitleStats: 算稱號用的統計（切/刮/磨次數、最佳倍率、圖鑑、籌碼、連磚、寶石）。
func (s *Store) TitleStats(userID int) (domain.TitleStats, error) {
	var ts domain.TitleStats
	err := s.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN action='cut' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN action='scratch' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN action='polish' THEN 1 ELSE 0 END),0),
		COALESCE(MAX(CASE WHEN action='cut' AND price>0 THEN CAST(payout AS REAL)/price ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN variety='彩蛋' THEN 1 ELSE 0 END),0)
		FROM stone_log WHERE user_id = ?`, userID).Scan(&ts.Cuts, &ts.Scratches, &ts.Polishes, &ts.BestMult, &ts.Gems)
	if err != nil {
		return ts, err
	}
	var chips, coll, streak, varieties int
	if err := s.db.QueryRow(`SELECT chips, collection_score, streak_brick FROM users WHERE id=?`, userID).
		Scan(&chips, &coll, &streak); err != nil {
		return ts, err
	}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM discovered WHERE user_id = ?`, userID).Scan(&varieties)
	ts.Chips, ts.Collection = chips, coll
	ts.BrickStreak = streak
	ts.Varieties = varieties
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM stone_log WHERE user_id=? AND quality='鑽石'`, userID).Scan(&ts.Diamonds)
	return ts, nil
}

// EquipTitle: 裝上一個稱號（空字串＝不顯示）。
func (s *Store) EquipTitle(userID int, title string) error {
	_, err := s.db.Exec(`UPDATE users SET title = ? WHERE id = ?`, title, userID)
	return err
}
