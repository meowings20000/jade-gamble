package store

import "database/sql"

// 獎勵兌換申請（2026-09-16）：玩家花籌碼申請獎勵（含 100 萬大獎與自訂獎勵），
// 一律進待審佇列，由管理員在控制臺通過或拒絕；通過才扣籌碼。

type RewardRequest struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Note      string `json:"note"`
	Cost      int    `json:"cost"`
	Status    string `json:"status"` // pending / approved / rejected
	AdminNote string `json:"admin_note"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) CreateRewardTx(tx *sql.Tx, userID int, title, note string, cost int) (int, error) {
	res, err := tx.Exec(`INSERT INTO reward_requests (user_id, title, note, cost, status)
		VALUES (?,?,?,?,'pending')`, userID, title, note, cost)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// RewardRequests: userID = 0 表示全部（管理員用）。
func (s *Store) RewardRequests(userID, limit int) ([]RewardRequest, error) {
	q := `SELECT r.id, r.user_id, COALESCE(u.username,''), r.title, r.note, r.cost, r.status,
		COALESCE(r.admin_note,''), r.created_at
		FROM reward_requests r LEFT JOIN users u ON u.id = r.user_id`
	args := []any{}
	if userID > 0 {
		q += ` WHERE r.user_id = ?`
		args = append(args, userID)
	}
	q += ` ORDER BY r.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RewardRequest{}
	for rows.Next() {
		var r RewardRequest
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Title, &r.Note, &r.Cost, &r.Status, &r.AdminNote, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RewardByID: 單筆（管理員審核用）。
func (s *Store) RewardByID(id int) (*RewardRequest, error) {
	var r RewardRequest
	err := s.db.QueryRow(`SELECT r.id, r.user_id, COALESCE(u.username,''), r.title, r.note, r.cost, r.status,
		COALESCE(r.admin_note,''), r.created_at FROM reward_requests r LEFT JOIN users u ON u.id = r.user_id
		WHERE r.id = ?`, id).Scan(&r.ID, &r.UserID, &r.Name, &r.Title, &r.Note, &r.Cost, &r.Status, &r.AdminNote, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
