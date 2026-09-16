package store

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"jade-gamble/backend/domain"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // sqlite: single writer
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			discord_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			avatar TEXT NOT NULL DEFAULT '',
			chips INTEGER NOT NULL DEFAULT 10000,
			collection_score INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			streak_brick INTEGER NOT NULL DEFAULT 0,
			daily_win_streak INTEGER NOT NULL DEFAULT 0,
			relief_used INTEGER NOT NULL DEFAULT 0,
			old_master_rescue INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			last_login_date TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS stones (
			id TEXT PRIMARY KEY,
			owner_id INTEGER,
			grade INTEGER NOT NULL,
			price INTEGER NOT NULL,
			seed INTEGER NOT NULL,
			quality INTEGER NOT NULL,
			variety INTEGER NOT NULL,
			crack_cells TEXT NOT NULL DEFAULT '[]',
			cracks_deep INTEGER NOT NULL DEFAULT 0,
			ink_hidden INTEGER NOT NULL DEFAULT 0,
			light_hint TEXT NOT NULL DEFAULT '',
			light_lie_rate REAL NOT NULL DEFAULT 0,
			egg TEXT NOT NULL DEFAULT '',
			bianhe INTEGER NOT NULL DEFAULT 0,
			origin TEXT NOT NULL DEFAULT 'shop',
			state TEXT NOT NULL DEFAULT 'owned',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS shelves (
			user_id INTEGER NOT NULL,
			grade INTEGER NOT NULL,
			slot INTEGER NOT NULL,
			stone_id TEXT,
			refreshed_today INTEGER NOT NULL DEFAULT 0,
			restock_date TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (user_id, grade, slot)
		)`,
		`CREATE TABLE IF NOT EXISTS inventory_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			item_key TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS active_buffs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			item_key TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS scratch_progress (
			stone_id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			revealed TEXT NOT NULL DEFAULT '{}',
			accumulated INTEGER NOT NULL DEFAULT 0,
			done INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS polish_progress (
			stone_id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			stage INTEGER NOT NULL DEFAULT 0,
			alive INTEGER NOT NULL DEFAULT 1,
			break_mod REAL NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS listings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			stone_id TEXT NOT NULL,
			seller_id INTEGER NOT NULL,
			ask_price INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			sold INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS discovered (
			user_id INTEGER NOT NULL,
			variety INTEGER NOT NULL,
			discovered_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, variety)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS hall_of_fame (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			stone_id TEXT NOT NULL,
			seed INTEGER NOT NULL,
			quality INTEGER NOT NULL,
			variety INTEGER NOT NULL,
			payout INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS npc_pool (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			stone_id TEXT NOT NULL,
			ask_price INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stones_owner ON stones(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_open ON listings(sold, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_npc_pool_user ON npc_pool(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// Additive migrations for pre-existing databases.
	if _, err := s.db.Exec(`ALTER TABLE stones ADD COLUMN origin TEXT NOT NULL DEFAULT 'shop'`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("migrate origin: %w", err)
	}
	if err := s.migrateYBoss(); err != nil {
		return fmt.Errorf("migrate yboss: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE scratch_progress ADD COLUMN layout TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("migrate scratch layout: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN relief_at TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("migrate relief_at: %w", err)
	}
	return nil
}

// ---------- users ----------

func (s *Store) GetUserByDiscordID(discordID string) (*domain.User, error) {
	row := s.db.QueryRow(`SELECT id, discord_id, username, avatar, chips, collection_score, title,
		streak_brick, daily_win_streak, relief_used, relief_at, old_master_rescue, last_login_date FROM users WHERE discord_id=?`, discordID)
	return scanUser(row)
}

func (s *Store) CreateUser(discordID, username, avatar string) (*domain.User, error) {
	res, err := s.db.Exec(`INSERT INTO users (discord_id, username, avatar) VALUES (?,?,?)`,
		discordID, username, avatar)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &domain.User{ID: int(id), DiscordID: discordID, Username: username, Avatar: avatar, Chips: 10000}, nil
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.DiscordID, &u.Username, &u.Avatar, &u.Chips, &u.CollectionScore,
		&u.Title, &u.StreakBrick, &u.DailyWinStreak, &u.ReliefUsed, &u.ReliefAt, &u.OldMasterRescue, &u.LastLoginDate)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUser(id int) (*domain.User, error) {
	row := s.db.QueryRow(`SELECT id, discord_id, username, avatar, chips, collection_score, title,
		streak_brick, daily_win_streak, relief_used, relief_at, old_master_rescue, last_login_date FROM users WHERE id=?`, id)
	return scanUser(row)
}

// UpdateChips atomically adds delta (may be negative) and returns new balance.
// Fails if the balance would go negative.
func (s *Store) UpdateChips(tx *sql.Tx, userID, delta int) (int, error) {
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

// Leaderboards: wealth and collection.
func (s *Store) Leaderboard(kind string, limit int) ([]domain.LeaderEntry, error) {
	var rows *sql.Rows
	var err error
	switch kind {
	case "wealth":
		rows, err = s.db.Query(`SELECT id, username, avatar, chips, 0 FROM users ORDER BY chips DESC LIMIT ?`, limit)
	case "collection":
		rows, err = s.db.Query(`SELECT id, username, avatar, collection_score, 0 FROM users ORDER BY collection_score DESC LIMIT ?`, limit)
	default:
		return nil, fmt.Errorf("unknown leaderboard kind %q", kind)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LeaderEntry
	for rows.Next() {
		var e domain.LeaderEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.Avatar, &e.Score, &e.Extra); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- stones ----------

func (s *Store) SaveStoneTx(tx *sql.Tx, st *domain.Stone) error {
	cells, _ := json.Marshal(st.CrackCells)
	_, err := tx.Exec(`INSERT INTO stones (id, owner_id, grade, price, seed, quality, variety,
		crack_cells, cracks_deep, ink_hidden, light_hint, light_lie_rate, egg, bianhe, state, origin)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET owner_id=excluded.owner_id, state=excluded.state`,
		st.ID, st.OwnerID, int(st.Grade), st.Price, int64(st.Seed), int(st.Quality), int(st.Variety),
		string(cells), b2i(st.CracksDeep), b2i(st.InkHidden), st.LightHint, st.LightLieRate, st.Egg, b2i(st.BianheGuaranteed),
		string(st.State), st.Origin)
	return err
}

func (s *Store) SaveStone(st *domain.Stone) error {
	cells, _ := json.Marshal(st.CrackCells)
	_, err := s.db.Exec(`INSERT INTO stones (id, owner_id, grade, price, seed, quality, variety,
		crack_cells, cracks_deep, ink_hidden, light_hint, light_lie_rate, egg, bianhe, state, origin)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET owner_id=excluded.owner_id, state=excluded.state`,
		st.ID, st.OwnerID, int(st.Grade), st.Price, int64(st.Seed), int(st.Quality), int(st.Variety),
		string(cells), b2i(st.CracksDeep), b2i(st.InkHidden), st.LightHint, st.LightLieRate, st.Egg, b2i(st.BianheGuaranteed),
		string(st.State), st.Origin)
	return err
}

func (s *Store) GetStone(id string) (*domain.Stone, error) {
	row := s.db.QueryRow(`SELECT id, owner_id, grade, price, seed, quality, variety,
		crack_cells, cracks_deep, ink_hidden, light_hint, light_lie_rate, egg, bianhe, state, origin
		FROM stones WHERE id=?`, id)
	return scanStone(row)
}

func scanStone(row *sql.Row) (*domain.Stone, error) {
	st := &domain.Stone{}
	var cells string
	var state string
	err := row.Scan(&st.ID, &st.OwnerID, &st.Grade, &st.Price, &st.Seed, &st.Quality, &st.Variety,
		&cells, &st.CracksDeep, &st.InkHidden, &st.LightHint, &st.LightLieRate, &st.Egg, &st.BianheGuaranteed, &state, &st.Origin)
	if err != nil {
		return nil, err
	}
	st.State = domain.StoneState(state)
	_ = json.Unmarshal([]byte(cells), &st.CrackCells)
	return st, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---------- sessions ----------

func (s *Store) CreateSession(userID int) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := crand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	if _, err := s.db.Exec(`INSERT INTO sessions (token, user_id) VALUES (?,?)`, token, userID); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) GetSession(token string) (int, error) {
	var uid int
	err := s.db.QueryRow(`SELECT user_id FROM sessions WHERE token=?`, token).Scan(&uid)
	return uid, err
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}

// ---------- discovery ----------

func (s *Store) DiscoverTx(tx *sql.Tx, userID int, v domain.ColorVariety) (first bool, err error) {
	res, err := tx.Exec(`INSERT OR IGNORE INTO discovered (user_id, variety) VALUES (?,?)`, userID, int(v))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *Store) DiscoveredList(userID int) ([]domain.ColorVariety, error) {
	rows, err := s.db.Query(`SELECT variety FROM discovered WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ColorVariety
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, domain.ColorVariety(v))
	}
	return out, rows.Err()
}

// ---------- hall of fame ----------

func (s *Store) AddHallOfFameTx(tx *sql.Tx, userID int, username string, st *domain.Stone, payout int) error {
	_, err := tx.Exec(`INSERT INTO hall_of_fame (user_id, username, stone_id, seed, quality, variety, payout)
		VALUES (?,?,?,?,?,?,?)`, userID, username, st.ID, int64(st.Seed), int(st.Quality), int(st.Variety), payout)
	return err
}

func (s *Store) HallOfFame(limit int) ([]domain.FameEntry, error) {
	rows, err := s.db.Query(`SELECT user_id, username, stone_id, seed, quality, variety, payout, created_at
		FROM hall_of_fame ORDER BY payout DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FameEntry
	for rows.Next() {
		var e domain.FameEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.StoneID, &e.Seed, &e.Quality, &e.Variety, &e.Payout, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) Now() time.Time { return time.Now().UTC() }
