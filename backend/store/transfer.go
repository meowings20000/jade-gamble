package store

import (
	"database/sql"
	"errors"
	"strings"

	"jade-gamble/backend/domain"
)

// Transfer 狀態
const (
	TransferPending   = "pending"   // 有條件，等對方接受
	TransferSent      = "sent"      // 無條件，即時到賬
	TransferAccepted  = "accepted"  // 對方接受
	TransferDeclined  = "declined"  // 對方拒絕（退還）
	TransferCancelled = "cancelled" // 發起人取消（退還）
)

// Transfer: 一筆轉賬。附條件時先扣發起人的籌碼（託管），對方接受才入賬。
type Transfer struct {
	ID        int    `json:"id"`
	FromID    int    `json:"from_id"`
	ToID      int    `json:"to_id"`
	FromName  string `json:"from"`
	ToName    string `json:"to"`
	Amount    int    `json:"amount"`
	Condition string `json:"condition"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// UserByUsername: 用名字找人（轉賬用）。
func (s *Store) UserByUsername(name string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("empty username")
	}
	row := s.db.QueryRow(`SELECT id, discord_id, username, avatar, chips FROM users WHERE username = ? COLLATE NOCASE`, name)
	var u domain.User
	if err := row.Scan(&u.ID, &u.DiscordID, &u.Username, &u.Avatar, &u.Chips); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateTransferTx: 開一筆轉賬。
func (s *Store) CreateTransferTx(tx *sql.Tx, fromID, toID, amount int, cond, status string) (int, error) {
	res, err := tx.Exec(`INSERT INTO transfers (from_id, to_id, amount, condition, status) VALUES (?,?,?,?,?)`,
		fromID, toID, amount, cond, status)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// TransferByID: 取一筆轉賬。
func (s *Store) TransferByID(id int) (*Transfer, error) {
	return s.scanTransfer(`SELECT t.id, t.from_id, t.to_id, f.username, u.username, t.amount, t.condition, t.status, t.created_at
		FROM transfers t JOIN users f ON f.id = t.from_id JOIN users u ON u.id = t.to_id
		WHERE t.id = ?`, id)
}

func (s *Store) scanTransfer(q string, args ...any) (*Transfer, error) {
	var t Transfer
	err := s.db.QueryRow(q, args...).Scan(&t.ID, &t.FromID, &t.ToID, &t.FromName, &t.ToName,
		&t.Amount, &t.Condition, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTransfers: 我的轉賬（待處理的收/發各一組，加上最近的歷史）。
func (s *Store) ListTransfers(uid, limit int) (incoming, outgoing, history []Transfer, err error) {
	incoming, err = s.queryTransfers(`t.to_id = ? AND t.status = 'pending' ORDER BY t.id DESC`, uid)
	if err != nil {
		return
	}
	outgoing, err = s.queryTransfers(`t.from_id = ? AND t.status = 'pending' ORDER BY t.id DESC`, uid)
	if err != nil {
		return
	}
	history, err = s.queryTransfers(`(t.from_id = ? OR t.to_id = ?) AND t.status != 'pending' ORDER BY t.id DESC LIMIT ?`, uid, uid, limit)
	return
}

func (s *Store) queryTransfers(where string, args ...any) ([]Transfer, error) {
	q := `SELECT t.id, t.from_id, t.to_id, f.username, u.username, t.amount, t.condition, t.status, t.created_at
		FROM transfers t JOIN users f ON f.id = t.from_id JOIN users u ON u.id = t.to_id
		WHERE ` + where
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transfer{}
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.FromID, &t.ToID, &t.FromName, &t.ToName,
			&t.Amount, &t.Condition, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ResolveTransferTx: 把一筆待處理的轉賬結案。
func (s *Store) ResolveTransferTx(tx *sql.Tx, id int, status string) error {
	_, err := tx.Exec(`UPDATE transfers SET status = ?, resolved_at = datetime('now') WHERE id = ?`, status, id)
	return err
}
