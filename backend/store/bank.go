package store

import (
	"database/sql"
	"errors"
	"jade-gamble/backend/domain"
)

// ---------- 喵喵錢莊 ----------

// ActiveLoan: 這個人還沒還完的貸款（沒有就 nil）。
func (s *Store) ActiveLoan(userID int) (*domain.Loan, error) {
	return s.scanLoan(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE user_id = ? AND status = 'active' ORDER BY id DESC LIMIT 1`, userID)
}

// LoanByID: 取一筆。
func (s *Store) LoanByID(id int) (*domain.Loan, error) {
	return s.scanLoan(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE id = ?`, id)
}

func (s *Store) scanLoan(q string, args ...any) (*domain.Loan, error) {
	var l domain.Loan
	err := s.db.QueryRow(q, args...).Scan(&l.ID, &l.UserID, &l.Principal, &l.Interest, &l.Hours,
		&l.DueAt, &l.Status, &l.Appeals, &l.Reason, &l.CreatedAt, &l.Rate, &l.Penalty)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateLoanTx: 放款（記帳；實際加籌碼由呼叫端在同一個 tx 做）。
func (s *Store) CreateLoanTx(tx *sql.Tx, userID, principal, interest, hours int, rate float64, penalty int, reason string) (int, error) {
	res, err := tx.Exec(`INSERT INTO loans (user_id, principal, interest, hours, rate, penalty, due_at, status, reason)
		VALUES (?,?,?,?,?,?, datetime('now', ?), 'active', ?)`,
		userID, principal, interest, hours, rate, penalty, fmtHours(hours), reason)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// LoanChatTx: 記錄對話（role = player / banker）。
func (s *Store) LoanChatTx(tx *sql.Tx, loanID int, role, content string) error {
	_, err := tx.Exec(`INSERT INTO loan_chat (loan_id, role, content) VALUES (?,?,?)`, loanID, role, content)
	return err
}

// LoanChat: 這筆貸款的對話紀錄（舊到新）。
func (s *Store) LoanChat(loanID int) ([]map[string]string, error) {
	rows, err := s.db.Query(`SELECT role, content FROM loan_chat WHERE loan_id = ? ORDER BY id`, loanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"role": role, "content": content})
	}
	return out, rows.Err()
}

// UpdateLoanTermsTx: 改條件（協商成功時）。
func (s *Store) UpdateLoanTermsTx(tx *sql.Tx, id, amount, hours int, rate float64, penalty int) error {
	_, err := tx.Exec(`UPDATE loans SET principal = ?, interest = ?, hours = ?, rate = ?, penalty = ?, due_at = datetime('now', ?) WHERE id = ?`,
		amount, domain.BankInterest(amount, rate), hours, rate, penalty, fmtHours(hours), id)
	return err
}

// BumpAppealTx: 申訴次數 +1。
func (s *Store) BumpAppealTx(tx *sql.Tx, id int) error {
	_, err := tx.Exec(`UPDATE loans SET appeals = appeals + 1 WHERE id = ?`, id)
	return err
}

// SetLoanStatusTx: 改狀態（repaid / defaulted）。
func (s *Store) SetLoanStatusTx(tx *sql.Tx, id int, status string) error {
	_, err := tx.Exec(`UPDATE loans SET status = ? WHERE id = ?`, status, id)
	return err
}

// OverdueLoans: 到期沒還的（逾期的第一時間要沒收一半財產）。
func (s *Store) OverdueLoans() ([]domain.Loan, error) {
	rows, err := s.db.Query(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE status = 'active' AND due_at < datetime('now') ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Loan{}
	for rows.Next() {
		var l domain.Loan
		if err := rows.Scan(&l.ID, &l.UserID, &l.Principal, &l.Interest, &l.Hours, &l.DueAt,
			&l.Status, &l.Appeals, &l.Reason, &l.CreatedAt, &l.Rate, &l.Penalty); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// OwnedStoneIDs: 這個人倉庫裡的石頭（沒收財產用）。
func (s *Store) OwnedStoneIDs(userID int) ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM stones WHERE owner_id = ? AND state = 'owned' ORDER BY price ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// LoanHistory: 這個人的借貸紀錄。
func (s *Store) LoanHistory(userID, limit int) ([]domain.Loan, error) {
	rows, err := s.db.Query(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Loan{}
	for rows.Next() {
		var l domain.Loan
		if err := rows.Scan(&l.ID, &l.UserID, &l.Principal, &l.Interest, &l.Hours, &l.DueAt,
			&l.Status, &l.Appeals, &l.Reason, &l.CreatedAt, &l.Rate, &l.Penalty); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func fmtHours(hours int) string {
	return "+" + itoa(hours) + " hours"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// SaveApplicationTx: 記錄一筆「被拒絕」的申請（玩家可以申訴）。
func (s *Store) SaveApplicationTx(tx *sql.Tx, userID, amount, hours int, rate float64, reason string) (int, error) {
	res, err := tx.Exec(`INSERT INTO loans (user_id, principal, interest, hours, rate, penalty, due_at, status, reason)
		VALUES (?,?,?,?,?,0, datetime('now', ?), 'denied', ?)`,
		userID, amount, domain.BankInterest(amount, rate), hours, rate, fmtHours(hours), reason)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// DeniedApplication: 最近一筆被拒絕、還能申訴的申請。
func (s *Store) DeniedApplication(userID int) (*domain.Loan, error) {
	return s.scanLoan(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE user_id = ? AND status = 'denied' AND appeals < 3 ORDER BY id DESC LIMIT 1`, userID)
}

// ActivateLoanTx: 申訴成功 → 放款（狀態轉 active）。
func (s *Store) ActivateLoanTx(tx *sql.Tx, id, amount, interest, hours int, rate float64, penalty int) error {
	_, err := tx.Exec(`UPDATE loans SET principal=?, interest=?, hours=?, rate=?, penalty=?, status='active',
		due_at = datetime('now', ?) WHERE id=?`, amount, interest, hours, rate, penalty, fmtHours(hours), id)
	return err
}

// OfferLoanTx: 存一筆「提議」（等玩家按接受）。
func (s *Store) OfferLoanTx(tx *sql.Tx, userID, amount, interest, hours int, rate float64, penalty int, reason string) (int, error) {
	res, err := tx.Exec(`INSERT INTO loans (user_id, principal, interest, hours, rate, penalty, due_at, status, reason)
		VALUES (?,?,?,?,?,?, datetime('now', ?), 'offered', ?)`,
		userID, amount, interest, hours, rate, penalty, fmtHours(hours), reason)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// PendingOffer: 最近一筆還在等玩家答覆的提議。
func (s *Store) PendingOffer(userID int) (*domain.Loan, error) {
	return s.scanLoan(`SELECT id, user_id, principal, interest, hours, due_at, status, appeals, reason, created_at, rate, penalty
		FROM loans WHERE user_id = ? AND status = 'offered' ORDER BY id DESC LIMIT 1`, userID)
}

// AcceptOfferTx: 玩家接受提議 → 放款（狀態轉 active、到期時間從現在算）。
func (s *Store) AcceptOfferTx(tx *sql.Tx, id, hours int) error {
	_, err := tx.Exec(`UPDATE loans SET status='active', due_at = datetime('now', ?) WHERE id = ?`, fmtHours(hours), id)
	return err
}

// SetLoanAppealsTx: 把申訴次數帶到新提議上（否則每輪都重置＝無限申訴）。
func (s *Store) SetLoanAppealsTx(tx *sql.Tx, id, n int) error {
	_, err := tx.Exec(`UPDATE loans SET appeals = ? WHERE id = ?`, n, id)
	return err
}

// LatestChat: 這名玩家「最近一筆」貸款的對話（不管狀態、是提議或被拒的都算）。
// 用途：/api/bank 顯示申訴來回——AI 可能把對話寫在提議那筆而不是生效中的貸款，
// 直接抓最近一筆最保險。
func (s *Store) LatestChat(userID int) ([]map[string]string, error) {
	var id int
	if err := s.db.QueryRow(`SELECT id FROM loans WHERE user_id=? ORDER BY id DESC LIMIT 1`, userID).Scan(&id); err != nil {
		return []map[string]string{}, nil
	}
	return s.LoanChat(id)
}

// CopyLoanChat: 把對話從舊貸款列搬到新列。
// 原因：AI 反提議（counter）會開一筆新的 loans 列，對話若留在舊列就會「看不到」。
func (s *Store) CopyLoanChat(fromID, toID int) error {
	if fromID == 0 || toID == 0 || fromID == toID {
		return nil
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM loan_chat WHERE loan_id=?`, toID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil // 新列自己已經有對話，不覆蓋
	}
	_, err := s.db.Exec(`INSERT INTO loan_chat (loan_id, role, content) SELECT ?, role, content FROM loan_chat WHERE loan_id=? ORDER BY id`, toID, fromID)
	return err
}
