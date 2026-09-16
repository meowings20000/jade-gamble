package store

import (
	"database/sql"
	"errors"

	"jade-gamble/backend/domain"
)

// FillShelfSlot places a stone in the first empty slot of the grade (non-tx).
func (s *Store) FillShelfSlot(userID int, grade domain.ShopGrade, stoneID string) error {
	res, err := s.db.Exec(`UPDATE shelves SET stone_id=? WHERE user_id=? AND grade=? AND slot=(
		SELECT MIN(slot) FROM shelves WHERE user_id=? AND grade=? AND stone_id IS NULL)`,
		stoneID, userID, int(grade), userID, int(grade))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("shelf full")
	}
	return nil
}

// FillShelfSlotTx is FillShelfSlot inside a transaction. CRITICAL: with
// MaxOpenConns(1) any non-tx call inside a tx closure deadlocks the pool.
func (s *Store) FillShelfSlotTx(tx *Tx, userID int, grade domain.ShopGrade, stoneID string) error {
	res, err := tx.Exec(`UPDATE shelves SET stone_id=? WHERE user_id=? AND grade=? AND slot=(
		SELECT MIN(slot) FROM shelves WHERE user_id=? AND grade=? AND stone_id IS NULL)`,
		stoneID, userID, int(grade), userID, int(grade))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("shelf full")
	}
	return nil
}

// UsernameTx reads a username inside a transaction (same deadlock rule).
func UsernameTx(tx *Tx, userID int) (string, error) {
	var name string
	err := tx.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return name, err
}
