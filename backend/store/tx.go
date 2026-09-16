package store

import (
	"database/sql"

	"jade-gamble/backend/domain"
)

// txWrap lets store methods be used with both raw sql.Tx and tests.
type txWrap = sql.Tx

// withTx runs fn inside a transaction, committing on nil error.
func (s *Store) withTx(fn func(tx *txWrap) error) error {
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

// withTxBool is withTx for functions returning a bool.
func (s *Store) withTxBool(fn func(tx *txWrap) (bool, error)) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	v, err := fn(tx)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	return v, tx.Commit()
}

// UpdateChipsTx atomically adds delta (may be negative) inside a tx.
// Fails with domain.ErrInsufficientChips if the balance would go negative.
func UpdateChipsTx(tx *txWrap, userID, delta int) (int, error) {
	res, err := tx.Exec(`UPDATE users SET chips = chips + ? WHERE id = ? AND chips + ? >= 0`, delta, userID, delta)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, domain.ErrInsufficientChips
	}
	var bal int
	if err := tx.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&bal); err != nil {
		return 0, err
	}
	return bal, nil
}
