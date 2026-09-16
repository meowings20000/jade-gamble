package store

// HasClassicPendingTx reports whether the user owns an unprocessed
// traditional-mode stone (origin='classic', state='owned').
func (s *Store) HasClassicPendingTx(tx *Tx, userID int) (bool, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM stones
		WHERE owner_id=? AND origin='classic' AND state='owned'`, userID).Scan(&n)
	return n > 0, err
}
