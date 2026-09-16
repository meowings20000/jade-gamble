package store

import (
	"jade-gamble/backend/domain"
)

// ClearShelfTx empties all slots of one grade.
func (s *Store) ClearShelfTx(tx *Tx, userID int, grade domain.ShopGrade) error {
	_, err := tx.Exec(`UPDATE shelves SET stone_id=NULL WHERE user_id=? AND grade=?`, userID, int(grade))
	return err
}

// IncRefreshTx bumps today's refresh counter (slot-0 row only).
func (s *Store) IncRefreshTx(tx *Tx, userID int, grade domain.ShopGrade, date string) error {
	_, err := tx.Exec(`UPDATE shelves SET refreshed_today = refreshed_today + 1
		WHERE user_id=? AND grade=? AND restock_date=? AND slot=0`, userID, int(grade), date)
	return err
}
