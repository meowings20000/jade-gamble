package store

import (
	"database/sql"
	"errors"
)

// YBoss sessions: one live polish gamble per user.
type YBossSession struct {
	ID       int
	UserID   int
	Stake    int
	Rung     int
	Finished bool
}

func (s *Store) migrateYBoss() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS yboss_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		stake INTEGER NOT NULL,
		rung INTEGER NOT NULL DEFAULT 0,
		finished INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	return err
}

func (s *Store) HasYBossSessionTx(userID int) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM yboss_sessions
		WHERE user_id=? AND finished=0`, userID).Scan(&n)
	return n > 0, err
}

func (s *Store) CreateYBossSessionTx(tx *Tx, userID, stake int) error {
	_, err := tx.Exec(`INSERT INTO yboss_sessions (user_id, stake, rung, finished) VALUES (?,?,0,0)`,
		userID, stake)
	return err
}

func (s *Store) GetYBossSession(userID int) (*YBossSession, error) {
	row := s.db.QueryRow(`SELECT id, user_id, stake, rung, finished FROM yboss_sessions
		WHERE user_id=? AND finished=0 ORDER BY id DESC LIMIT 1`, userID)
	var sess YBossSession
	var fin int
	err := row.Scan(&sess.ID, &sess.UserID, &sess.Stake, &sess.Rung, &fin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.Finished = fin == 1
	return &sess, nil
}

func (s *Store) SetYBossRungTx(tx *Tx, id, rung int) error {
	_, err := tx.Exec(`UPDATE yboss_sessions SET rung=? WHERE id=?`, rung, id)
	return err
}

func (s *Store) FinishYBossSessionTx(tx *Tx, id int) error {
	_, err := tx.Exec(`UPDATE yboss_sessions SET finished=1 WHERE id=?`, id)
	return err
}
