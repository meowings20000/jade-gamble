package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// ---------- 竞标市场 ----------

func (a *API) marketList(w http.ResponseWriter, r *http.Request) error {
	listings, err := a.Store.OpenListings(50)
	if err != nil {
		return err
	}
	if listings == nil {
		listings = []store.Listing{}
	}
	writeJSON(w, 200, map[string]any{"listings": publicListings(listings)})
	return nil
}

// publicListings hides seller identity details beyond the username.
func publicListings(in []store.Listing) []map[string]any {
	out := []map[string]any{}
	for _, l := range in {
		out = append(out, map[string]any{
			"id": l.ID, "stone_id": l.StoneID, "seller": l.SellerName,
			"ask_price": l.AskPrice, "grade": l.Grade, "seed": l.Seed,
			"light_hint": l.LightHint, "window_desc": l.WindowDesc,
		})
	}
	return out
}

func (a *API) marketListStone(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID  string `json:"stone_id"`
		AskPrice int    `json:"ask_price"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.AskPrice < 1 {
		return errors.New("出价必须为正")
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil || st.OwnerID != uid || st.State != domain.StateOwned {
		return store.ErrNotOwned
	}
	if st.Origin == "classic" {
		return errors.New("传统石不能挂卖")
	}
	// 5% listing fee (sink)
	fee := body.AskPrice / 20
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if _, err := store.UpdateChipsTx(tx, uid, -fee); err != nil {
			return err
		}
		if err := a.Store.CreateListingTx(tx, st.ID, uid, body.AskPrice); err != nil {
			return err
		}
		return a.Store.SetStoneStateTx(tx, st.ID, domain.StateListed, uid)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "fee": fee})
	return nil
}

func (a *API) marketBuy(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		ListingID int `json:"listing_id"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	l, err := a.Store.GetListing(body.ListingID)
	if err != nil {
		return err
	}
	if l.SellerID == uid {
		return errors.New("不能买自己挂的单")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if err := a.Store.BuyListingTx(tx, l.ID, uid); err != nil {
			return err
		}
		b, err := store.UpdateChipsTx(tx, uid, -l.AskPrice)
		if err != nil {
			return err
		}
		bal = b
		// seller receives price minus 5% commission
		_, err = tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`, l.AskPrice-l.AskPrice/20, l.SellerID)
		if err != nil {
			return err
		}
		return a.Store.SetStoneStateTx(tx, l.StoneID, domain.StateOwned, uid)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal, "stone_id": l.StoneID})
	return nil
}

// ---------- 兑换所 ----------

func (a *API) exchangeView(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	catalog := []map[string]any{}
	for _, it := range domain.ExchangeCatalog {
		price := it.Price
		if it.Key == "insurance" {
			price = 0 // computed per stone at purchase time
		}
		catalog = append(catalog, map[string]any{
			"key": it.Key, "name": it.Name, "price": price,
			"kind": it.Kind, "description": it.Description,
		})
	}
	writeJSON(w, 200, map[string]any{"chips": u.Chips, "catalog": catalog})
	return nil
}

func (a *API) exchangeBuy(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	it, found := domain.ExchangeItemByKey(body.Key)
	if !found {
		return errors.New("兌換所沒有這個商品")
	}
	price := it.Price
	if it.Key == "frenzy_ticket" {
		// grants 10 casual stones immediately
		var bal int
		total := 0
		pays := []int{}
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			b, err := store.UpdateChipsTx(tx, uid, -price)
			if err != nil {
				return err
			}
			bal = b
			for i := 0; i < 10; i++ {
				p := domain.FrenzyTicketPayout(domain.RandSource)
				pays = append(pays, p)
				total += p
			}
			total2 := total
			_, err = tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`, total2, uid)
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"payouts": pays, "total": total, "chips": bal + total})
		return nil
	}
	if it.Kind == "buff" {
		var hours int
		switch it.Key {
		case "light_master":
			hours = 24
		case "golden_eye", "polish_touch":
			hours = 1
		default:
			return errors.New("unknown buff")
		}
		exp := time.Now().UTC().Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			if _, err := store.UpdateChipsTx(tx, uid, -price); err != nil {
				return err
			}
			return a.Store.AddBuffTx(tx, uid, it.Key, exp)
		}); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "expires": exp})
		return nil
	}
	// consumable / cosmetic → inventory
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if _, err := store.UpdateChipsTx(tx, uid, -price); err != nil {
			return err
		}
		return a.Store.AddItemTx(tx, uid, it.Key)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true})
	return nil
}

// ---------- 破产救济 ----------

func (a *API) relief(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Option string `json:"option"` // "chips" | "ticket"
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	if u.Chips > 0 {
		return errors.New("你還有籌碼，不需要救濟")
	}
	// 3-hour cooldown between reliefs (relief_used keeps a counter for stats)
	if u.ReliefAt != "" {
		last, err := time.Parse(time.RFC3339, u.ReliefAt)
		if err == nil && time.Since(last) < 3*time.Hour {
			left := 3*time.Hour - time.Since(last)
			return fmt.Errorf("救濟冷卻中，還剩 %d 分鐘", int(left.Minutes())+1)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if body.Option == "ticket" {
		var pays []int
		total := 0
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			for i := 0; i < 10; i++ {
				p := domain.FrenzyTicketPayout(domain.RandSource)
				pays = append(pays, p)
				total += p
			}
			_, err := tx.Exec(`UPDATE users SET chips = chips + ?, relief_used = relief_used + 1, relief_at = ? WHERE id=?`,
				total+domain.ReliefAltChips, now, uid)
			return err
		}); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"payouts": pays, "chips_given": domain.ReliefAltChips, "total": total})
		return nil
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		_, err := tx.Exec(`UPDATE users SET chips = chips + ?, relief_used = relief_used + 1, relief_at = ? WHERE id=?`,
			domain.ReliefChips, now, uid)
		return err
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"chips_given": domain.ReliefChips})
	return nil
}

// ---------- 榜 ----------

func (a *API) leaderboard(w http.ResponseWriter, r *http.Request) error {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "wealth"
	}
	entries, err := a.Store.Leaderboard(kind, 50)
	if err != nil {
		return err
	}
	if entries == nil {
		entries = []domain.LeaderEntry{}
	}
	writeJSON(w, 200, map[string]any{"kind": kind, "entries": entries})
	return nil
}

func (a *API) collection(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	disc, err := a.Store.DiscoveredList(uid)
	if err != nil {
		return err
	}
	have := map[int]bool{}
	for _, v := range disc {
		have[int(v)] = true
	}
	catalog := []map[string]any{}
	for v := domain.Violet; v <= domain.ImperialGreen; v++ {
		catalog = append(catalog, map[string]any{
			"variety": int(v), "name": v.Name(), "score": v.CollectionScore(),
			"discovered": have[int(v)],
		})
	}
	writeJSON(w, 200, map[string]any{
		"score": u.CollectionScore, "varieties": catalog,
	})
	return nil
}

func (a *API) hallOfFame(w http.ResponseWriter, r *http.Request) error {
	entries, err := a.Store.HallOfFame(20)
	if err != nil {
		return err
	}
	if entries == nil {
		entries = []domain.FameEntry{}
	}
	writeJSON(w, 200, map[string]any{"entries": entries})
	return nil
}

var _ = fmt.Sprintf
