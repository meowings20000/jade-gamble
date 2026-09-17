package store

import "database/sql"

// 寶石圖鑒（2026-09-17 用戶要求）：記錄玩家切到過的彩蛋寶石。
// 只要「切到過」就永久記錄，重複切到會累加次數。

func (s *Store) EnsureGemTable() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS gem_collection (
		user_id  INTEGER NOT NULL,
		gem_key  TEXT    NOT NULL,
		count    INTEGER NOT NULL DEFAULT 0,
		first_at TEXT    NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (user_id, gem_key))`)
	return err
}

// RecordGemTx: 切到彩蛋時累加（Tx-aware：跟切石的其它寫入同一個交易）。
func (s *Store) RecordGemTx(tx *sql.Tx, userID int, key string) error {
	if key == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO gem_collection (user_id, gem_key, count) VALUES (?,?,1)
		ON CONFLICT(user_id, gem_key) DO UPDATE SET count = count + 1`, userID, key)
	return err
}

// GemOwned: 圖鑒裡的一格。
type GemOwned struct {
	Key     string `json:"key"`
	Count   int    `json:"count"`
	FirstAt string `json:"first_at"`
}

// GemCollection: 這個人收集到的寶石（key → 次數／首次時間）。
func (s *Store) GemCollection(userID int) (map[string]GemOwned, error) {
	rows, err := s.db.Query(`SELECT gem_key, count, first_at FROM gem_collection WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]GemOwned{}
	for rows.Next() {
		var g GemOwned
		if err := rows.Scan(&g.Key, &g.Count, &g.FirstAt); err != nil {
			return nil, err
		}
		out[g.Key] = g
	}
	return out, rows.Err()
}
