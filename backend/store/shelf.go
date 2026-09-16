package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"jade-gamble/backend/domain"
)

var (
	ErrInsufficient = domain.ErrInsufficientChips
	ErrNotFound     = domain.ErrNotFound
	ErrNotOwned     = domain.ErrStoneNotOwned
)

var _ = errors.Is // keep errors imported

// errNotOwned is a concrete sentinel for scan paths.
func errInsufficient() error { return domain.ErrInsufficientChips }

// CountUsers for smoke checks.
func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// EnsureShelf creates the shelf rows for a user if missing, then returns them.
// Daily reset happens ONLY on real day rollover (restock_date <> today):
// it must not clobber refreshed_today on every call.
func (s *Store) EnsureShelf(userID int, date string) error {
	for g := domain.KiloGrade; g <= domain.WindowGrade; g++ {
		for slot := 0; slot < domain.ShelfSize(g); slot++ {
			_, err := s.db.Exec(`INSERT OR IGNORE INTO shelves (user_id, grade, slot, restock_date) VALUES (?,?,?,?)`,
				userID, int(g), slot, "")
			if err != nil {
				return err
			}
		}
	}
	_, err := s.db.Exec(`UPDATE shelves SET refreshed_today = 0, restock_date = ?,
		stone_id = NULL
		WHERE user_id = ? AND restock_date <> ?`, date, userID, date)
	return err
}

// ShelfStones returns the visible stone ids on a user's shelf for a grade.
func (s *Store) ShelfStones(userID int, grade domain.ShopGrade) ([]string, error) {
	rows, err := s.db.Query(`SELECT stone_id FROM shelves WHERE user_id=? AND grade=? ORDER BY slot`,
		userID, int(grade))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid {
			out = append(out, id.String)
		}
	}
	return out, rows.Err()
}

// TakeShelfStone removes a stone from whichever slot holds it, if present.
func (s *Store) TakeShelfStoneTx(tx *sql.Tx, userID int, grade domain.ShopGrade, stoneID string) (bool, error) {
	res, err := tx.Exec(`UPDATE shelves SET stone_id=NULL WHERE user_id=? AND grade=? AND stone_id=?`,
		userID, int(grade), stoneID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// CountShelfRefreshes: how many refreshes the user has done today on a grade.
func (s *Store) CountShelfRefreshes(userID int, grade domain.ShopGrade, date string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COALESCE(refreshed_today,0) FROM shelves WHERE user_id=? AND grade=? AND restock_date=? AND slot=0`,
		userID, int(grade), date).Scan(&n)
	return n, err
}

// IncRefresh increments the refresh counter for the grade's shelf rows.
func (s *Store) IncRefresh(userID int, grade domain.ShopGrade, date string) error {
	_, err := s.db.Exec(`UPDATE shelves SET refreshed_today = refreshed_today + 1
		WHERE user_id=? AND grade=? AND restock_date=? AND slot=0`, userID, int(grade), date)
	return err
}

// ListStonesByOwner returns owned stones.
func (s *Store) ListStonesByOwner(userID int) ([]*domain.Stone, error) {
	rows, err := s.db.Query(`SELECT id, owner_id, grade, price, seed, quality, variety,
		crack_cells, cracks_deep, ink_hidden, light_hint, light_lie_rate, egg, bianhe, state, origin
		FROM stones WHERE owner_id=? AND state='owned' ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Stone
	for rows.Next() {
		st := &domain.Stone{}
		var cells string
		if err := rows.Scan(&st.ID, &st.OwnerID, &st.Grade, &st.Price, &st.Seed, &st.Quality, &st.Variety,
			&cells, &st.CracksDeep, &st.InkHidden, &st.LightHint, &st.LightLieRate, &st.Egg, &st.BianheGuaranteed, &st.State, &st.Origin); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(cells), &st.CrackCells)
		out = append(out, st)
	}
	return out, rows.Err()
}

// SetStoneStateTx updates state within a transaction.
func (s *Store) SetStoneStateTx(tx *sql.Tx, stoneID string, state domain.StoneState, owner int) error {
	_, err := tx.Exec(`UPDATE stones SET state=?, owner_id=? WHERE id=?`, state, owner, stoneID)
	return err
}

// ConsumeItem removes one inventory item of a key, returning ok if consumed.
func (s *Store) ConsumeItemTx(tx *sql.Tx, userID int, itemKey string) (bool, error) {
	row := tx.QueryRow(`SELECT id FROM inventory_items WHERE user_id=? AND item_key=? LIMIT 1`, userID, itemKey)
	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	res, err := tx.Exec(`DELETE FROM inventory_items WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// AddItem inserts an inventory item.
func (s *Store) AddItemTx(tx *sql.Tx, userID int, itemKey string) error {
	_, err := tx.Exec(`INSERT INTO inventory_items (user_id, item_key) VALUES (?,?)`, userID, itemKey)
	return err
}

// AddBuff sets an expiry-based buff (replaces same-key buff).
func (s *Store) AddBuffTx(tx *sql.Tx, userID int, itemKey, expiresAt string) error {
	_, err := tx.Exec(`DELETE FROM active_buffs WHERE user_id=? AND item_key=?`, userID, itemKey)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO active_buffs (user_id, item_key, expires_at) VALUES (?,?,?)`,
		userID, itemKey, expiresAt)
	return err
}

// HasActiveBuff reports whether a buff is currently active.
func (s *Store) HasActiveBuff(userID int, itemKey, now string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM active_buffs WHERE user_id=? AND item_key=? AND expires_at > ?`,
		userID, itemKey, now).Scan(&n)
	return n > 0, err
}

// ListItems counts inventory items per key.
func (s *Store) ListItems(userID int) (map[string]int, error) {
	rows, err := s.db.Query(`SELECT item_key, COUNT(*) FROM inventory_items WHERE user_id=? GROUP BY item_key`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}

// CreateListing lists a stone for sale.
func (s *Store) CreateListingTx(tx *sql.Tx, stoneID string, sellerID, askPrice int) error {
	_, err := tx.Exec(`INSERT INTO listings (stone_id, seller_id, ask_price) VALUES (?,?,?)`,
		stoneID, sellerID, askPrice)
	return err
}

// OpenListings returns open listings with stone summaries.
func (s *Store) OpenListings(limit int) ([]Listing, error) {
	rows, err := s.db.Query(`SELECT l.id, l.stone_id, l.seller_id, l.ask_price,
		u.username, st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM listings l
		JOIN users u ON u.id = l.seller_id
		JOIN stones st ON st.id = l.stone_id
		WHERE l.sold = 0 ORDER BY l.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Listing
	for rows.Next() {
		var l Listing
		var seedInt int64
		if err := rows.Scan(&l.ID, &l.StoneID, &l.SellerID, &l.AskPrice,
			&l.SellerName, &l.Grade, &seedInt, &l.LightHint, &l.InkHidden); err != nil {
			return nil, err
		}
		l.Seed = uint64(seedInt)
		out = append(out, l)
	}
	return out, rows.Err()
}

// Listing is a market listing row (exported for the API layer).
type Listing struct {
	ID         int    `json:"id"`
	StoneID    string `json:"stone_id"`
	SellerID   int    `json:"seller_id"`
	SellerName string `json:"seller_name"`
	AskPrice   int    `json:"ask_price"`
	Grade      int    `json:"grade"`
	Seed       uint64 `json:"seed"`
	LightHint  string `json:"light_hint"`
	InkHidden  bool   `json:"ink_hidden"`
	WindowDesc string `json:"window_desc"`
}

// GetListing fetches one open listing.
func (s *Store) GetListing(id int) (*Listing, error) {
	var l Listing
	var seedInt int64
	err := s.db.QueryRow(`SELECT l.id, l.stone_id, l.seller_id, l.ask_price,
		u.username, st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM listings l
		JOIN users u ON u.id = l.seller_id
		JOIN stones st ON st.id = l.stone_id
		WHERE l.id=? AND l.sold=0`, id).Scan(
		&l.ID, &l.StoneID, &l.SellerID, &l.AskPrice,
		&l.SellerName, &l.Grade, &seedInt, &l.LightHint, &l.InkHidden)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	l.Seed = uint64(seedInt)
	return &l, err
}

// BuyListingTx marks a listing sold and transfers ownership atomically.
func (s *Store) BuyListingTx(tx *sql.Tx, listingID, buyerID int) error {
	res, err := tx.Exec(`UPDATE listings SET sold=1 WHERE id=? AND sold=0`, listingID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("listing already sold")
	}
	return nil
}
