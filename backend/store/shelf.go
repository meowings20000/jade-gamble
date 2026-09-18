package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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
// EnsureShelf: 建立貨架列；跨日時把整排清空（隔天免費補貨一次的起點）。
// 回傳 newDay = 這次呼叫是否跨日（呼叫端據此決定要不要真的生石頭）。
func (s *Store) EnsureShelf(userID int, date, bucket string) (bool, error) {
	for g := domain.KiloGrade; g <= domain.WindowGrade; g++ {
		for slot := 0; slot < domain.ShelfSize(g); slot++ {
			_, err := s.db.Exec(`INSERT OR IGNORE INTO shelves (user_id, grade, slot, restock_date) VALUES (?,?,?,?)`,
				userID, int(g), slot, "")
			if err != nil {
				return false, err
			}
		}
	}
	// 每 12 小時重置「刷新價格階梯」（次數歸零），跟每日補貨是兩件事
	if _, err := s.db.Exec(`UPDATE shelves SET refreshed_today = 0, refresh_bucket = ?
		WHERE user_id = ? AND refresh_bucket <> ?`, bucket, userID, bucket); err != nil {
		return false, err
	}
	res, err := s.db.Exec(`UPDATE shelves SET refreshed_today = 0, restock_date = ?,
		stone_id = NULL
		WHERE user_id = ? AND restock_date <> ?`, date, userID, date)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
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
		COALESCE(u.username, '礦區直送'), st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM listings l
		LEFT JOIN users u ON u.id = l.seller_id
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

// ---------- 拍賣機器人（收料 bot）----------

// BotCandidate: 一張放太久沒人標的掛單，連同它的喊價。
type BotCandidate struct {
	ListingID int
	StoneID   string
	SellerID  int
	AskPrice  int
	AgeHours  float64
}

// StaleListings: 玩家掛單（seller_id != 0）放置超過 staleHours 小時還沒賣掉的。
func (s *Store) StaleListings(staleMinutes float64, limit int) ([]BotCandidate, error) {
	rows, err := s.db.Query(`SELECT l.id, l.stone_id, l.seller_id, l.ask_price,
		(julianday('now') - julianday(l.created_at)) * 1440.0 AS age_mins
		FROM listings l
		WHERE l.sold = 0 AND l.seller_id != 0
		  AND (julianday('now') - julianday(l.created_at)) * 1440.0 >= ?
		ORDER BY l.created_at ASC LIMIT ?`, staleMinutes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotCandidate{}
	for rows.Next() {
		var c BotCandidate
		if err := rows.Scan(&c.ListingID, &c.StoneID, &c.SellerID, &c.AskPrice, &c.AgeHours); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RecordBotBuyTx: 記一筆 bot 成交，讓市場看得到「誰剛才來收料」。
func (s *Store) RecordBotBuyTx(tx *sql.Tx, stoneID string, sellerID int, botKey, botName string, price int, note string) error {
	_, err := tx.Exec(`INSERT INTO bot_buys (stone_id, seller_id, bot_key, bot_name, price, note) VALUES (?,?,?,?,?,?)`,
		stoneID, sellerID, botKey, botName, price, note)
	return err
}

// BotBuy: 一筆 bot 收料紀錄。
type BotBuy struct {
	StoneID string `json:"stone_id"`
	BotName string `json:"bot"`
	Price   int    `json:"price"`
	AgeMins int    `json:"age_mins"`
	Note    string `json:"note"`
}

// RecentBotBuys: 最近的 bot 收料紀錄（市場頁顯示用）。
func (s *Store) RecentBotBuys(limit int) ([]BotBuy, error) {
	rows, err := s.db.Query(`SELECT stone_id, bot_name, price, note,
		CAST((julianday('now') - julianday(created_at)) * 1440 AS INTEGER)
		FROM bot_buys ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BotBuy{}
	for rows.Next() {
		var b BotBuy
		if err := rows.Scan(&b.StoneID, &b.BotName, &b.Price, &b.Note, &b.AgeMins); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BackdateListing: 測試用——把掛單時間往前挪，模擬「放很久沒人標」。
func (s *Store) BackdateListing(id int, hours float64) error {
	_, err := s.db.Exec(`UPDATE listings SET created_at = datetime('now', ?) WHERE id=?`,
		fmt.Sprintf("-%f hours", hours), id)
	return err
}

// CountNPCPool: stones left in THIS user's private 礦區直送 pool.
func (s *Store) CountNPCPool(userID int) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM npc_pool WHERE user_id=?`, userID).Scan(&n)
	return n, err
}

// CreateNPCPoolStone: a fair-roll stone listed privately for one user.
func (s *Store) CreateNPCPoolStone(userID int, stoneID string, ask int) error {
	_, err := s.db.Exec(`INSERT INTO npc_pool (user_id, stone_id, ask_price) VALUES (?,?,?)`,
		userID, stoneID, ask)
	return err
}

// NPCPoolListings: this user's pool entries joined to their stones.
func (s *Store) NPCPoolListings(userID int) ([]Listing, error) {
	rows, err := s.db.Query(`SELECT np.id, np.stone_id, 0, np.ask_price,
		st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM npc_pool np
		JOIN stones st ON st.id = np.stone_id
		WHERE np.user_id=? ORDER BY np.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Listing
	for rows.Next() {
		var l Listing
		var seedInt int64
		if err := rows.Scan(&l.ID, &l.StoneID, &l.SellerID, &l.AskPrice,
			&l.Grade, &seedInt, &l.LightHint, &l.InkHidden); err != nil {
			return nil, err
		}
		l.Seed = uint64(seedInt)
		l.SellerName = "礦區直送"
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetNPCPoolListing: one pool entry for this user (buy path).
func (s *Store) GetNPCPoolListing(userID, poolID int) (*Listing, error) {
	var l Listing
	var seedInt int64
	err := s.db.QueryRow(`SELECT np.id, np.stone_id, 0, np.ask_price,
		st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM npc_pool np
		JOIN stones st ON st.id = np.stone_id
		WHERE np.id=? AND np.user_id=?`, poolID, userID).Scan(
		&l.ID, &l.StoneID, &l.SellerID, &l.AskPrice,
		&l.Grade, &seedInt, &l.LightHint, &l.InkHidden)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l.Seed = uint64(seedInt)
	l.SellerName = "礦區直送"
	return &l, nil
}

// DeleteNPCPoolStoneTx removes a bought pool entry atomically.
func (s *Store) DeleteNPCPoolStoneTx(tx *sql.Tx, poolID, userID int) error {
	res, err := tx.Exec(`DELETE FROM npc_pool WHERE id=? AND user_id=?`, poolID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("pool listing already sold")
	}
	return nil
}

// GetListing fetches one open listing.
func (s *Store) GetListing(id int) (*Listing, error) {
	var l Listing
	var seedInt int64
	err := s.db.QueryRow(`SELECT l.id, l.stone_id, l.seller_id, l.ask_price,
		COALESCE(u.username, '礦區直送'), st.grade, st.seed, st.light_hint, st.ink_hidden
		FROM listings l
		LEFT JOIN users u ON u.id = l.seller_id
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

// ---------- 打燈紀錄 ----------

// IsLit: 這顆石頭有沒有人付費打過燈（打過的才看得到描述）。
func (s *Store) IsLit(stoneID string) bool {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM lit_stones WHERE stone_id = ?`, stoneID).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// MarkLit: 記下這顆石頭打過燈。
func (s *Store) MarkLit(stoneID string, userID int) error {
	_, err := s.db.Exec(`INSERT INTO lit_stones (stone_id, user_id) VALUES (?,?)
		ON CONFLICT(stone_id) DO NOTHING`, stoneID, userID)
	return err
}

// EquippedFrame: 玩家有沒有金的／帝王的頭像框（兌換所的裝飾品）。
func (s *Store) EquippedFrame(userID int) string {
	for _, key := range []string{"frame_rainbow", "frame_ink", "frame_imperial", "frame_violet", "frame_gold", "frame_cat"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_items WHERE user_id = ? AND item_key = ?`, userID, key).Scan(&n); err == nil && n > 0 {
			return key
		}
	}
	return ""
}

// ShelfItemCount: 這個檔位的貨架上還有幾顆石頭（賣光時刷新應該免費）。
func (s *Store) ShelfItemCount(userID int, grade domain.ShopGrade) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(stone_id) FROM shelves WHERE user_id=? AND grade=?`, userID, int(grade)).Scan(&n)
	return n, err
}

// OwnsItem: 玩家有沒有這件收藏品。
func (s *Store) OwnsItem(userID int, key string) bool {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM inventory_items WHERE user_id = ? AND item_key = ?`, userID, key).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// EquippedTheme: 擁有的賭桌配色（貴的優先），沒有就空字串＝預設。
func (s *Store) EquippedTheme(userID int) string {
	for _, key := range []string{"theme_gold", "theme_ink", "theme_violet", "theme_jade"} {
		if s.OwnsItem(userID, key) {
			return key
		}
	}
	return ""
}

// HasRevealFX: 有沒有開箱彩帶特效。
func (s *Store) HasRevealFX(userID int) bool {
	return s.OwnsItem(userID, "fx_confetti")
}

// EquippedBubble: 擁有的聊天泡泡框（貴的優先），沒有回空字串。
func (s *Store) EquippedBubble(userID int) string {
	for _, key := range []string{"bubble_gold", "bubble_violet", "bubble_pink"} {
		if s.OwnsItem(userID, key) {
			return key
		}
	}
	return ""
}
