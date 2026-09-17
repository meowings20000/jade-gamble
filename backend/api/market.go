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
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	// 礦區直送: each player has a PRIVATE pool of fair-roll stones, so
	// nobody can infer where the good stones are by watching others buy.
	a.restockNPC(uid)
	a.SettleMarketBots()
	playerListings, err := a.Store.OpenListings(50)
	if err != nil {
		return err
	}
	pool, err := a.Store.NPCPoolListings(uid)
	if err != nil {
		return err
	}
	out := append(a.publicListings(playerListings), a.publicListings(pool)...)
	recent, err := a.Store.RecentBotBuys(6)
	if err != nil {
		recent = nil
	}
	writeJSON(w, 200, map[string]any{"listings": out, "bot_buys": recent})
	return nil
}

// SettleMarketBots: 沒人標的料，讓拍賣 bot 來收。
//
// 每隻 bot 有自己放單多久才出手的耐心（Sticky）與出價眼力（Eye × 真值）。
// 走漏眼的（賭鬼阿明 1.35×）常常出得比真值高；精明的（老周 0.72×）只撿便宜。
// 賣家照樣拿 95%（扣 5% 手續費），石頭被 bot 收走後不再流通。
func (a *API) SettleMarketBots() {
	for _, b := range domain.MarketBots {
		cands, err := a.Store.StaleListings(b.StickyMins, b.Appetite)
		if err != nil {
			continue
		}
		for _, c := range cands {
			st, err := a.Store.GetStone(c.StoneID)
			if err != nil {
				continue
			}
			if !domain.BotBuys(b, st, c.AskPrice) {
				continue // 這隻 bot 覺得不值這個價
			}
			err = a.Store.WithTx(func(tx *store.Tx) error {
				if err := a.Store.BuyListingTx(tx, c.ListingID, 0); err != nil {
					return err
				}
				// 賣家收錢（95%，與真人交易同規則）
				if _, err := tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`,
					c.AskPrice-c.AskPrice/20, c.SellerID); err != nil {
					return err
				}
				if err := a.Store.SetStoneStateTx(tx, c.StoneID, domain.StateSold, 0); err != nil {
					return err
				}
				return a.Store.RecordBotBuyTx(tx, c.StoneID, c.SellerID, b.Key, b.Name, c.AskPrice, domain.BotQuip(b, domain.RandSource))
			})
			if err != nil {
				continue
			}
			// 一個 bot 一次只收這麼多件
			b.Appetite--
			if b.Appetite <= 0 {
				break
			}
		}
	}
}

// npcRestockTarget: how many NPC stones each player's pool keeps.
const npcRestockTarget = 6

// restockNPC tops up THIS player's private pool with fair stones (same
// quality distribution as shop shelves) listed at a 25%~60% premium. Truth
// is fixed at generation; the buyer gambles on grade + hint, same as shop.
func (a *API) restockNPC(uid int) {
	n, err := a.Store.CountNPCPool(uid)
	if err != nil || n >= npcRestockTarget {
		return
	}
	for i := n; i < npcRestockTarget; i++ {
		// Weighted toward affordable grades (公斤料 50% / 表現料 33% / 開窗料 17%)
		// so a fresh 10k player can actually bid. High grades still appear.
		roll := domain.RandSource.Intn(6)
		grade := domain.KiloGrade
		switch {
		case roll < 3:
			grade = domain.KiloGrade
		case roll < 5:
			grade = domain.FeatureGrade
		default:
			grade = domain.WindowGrade
		}
		st := domain.GenerateStone(grade, domain.RandSource)
		st.State = domain.StateListed // NPC stock: listed, not owned
		st.Origin = "market"          // 礦區直送
		premium := 1.25 + domain.RandSource.Float64()*0.35
		ask := int(float64(st.Price) * premium)
		if ask < 1 {
			ask = 1
		}
		if err := a.Store.SaveStone(st); err != nil {
			return
		}
		if err := a.Store.CreateNPCPoolStone(uid, st.ID, ask); err != nil {
			return
		}
	}
}

// npcIDOffset separates pool IDs from player-listing IDs in the public
// payload so a single "listing_id" field can address both.
const npcIDOffset = 1_000_000_000

func isNPCID(id int) bool { return id >= npcIDOffset }

// publicListings: 拍賣匿名 — seller identity never leaves the server.
func (a *API) publicListings(in []store.Listing) []map[string]any {
	out := []map[string]any{}
	for _, l := range in {
		pubID := l.ID
		if l.SellerID == 0 {
			pubID += npcIDOffset
		}
		// 沒打過燈的掛單只有模糊描述（打過燈的賣家才會帶報告）
		hint := domain.DescribeFreeByGrade(domain.ShopGrade(l.Grade))
		if a.Store.IsLit(l.StoneID) {
			hint = l.LightHint
		}
		out = append(out, map[string]any{
			"id": pubID, "stone_id": l.StoneID,
			"ask_price": l.AskPrice, "grade": l.Grade, "seed": l.Seed,
			"light_hint": hint, "lit": a.Store.IsLit(l.StoneID),
			"npc": l.SellerID == 0,
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
	var bal int
	if isNPCID(body.ListingID) {
		// 礦區直送: private pool entry, chips sink into the mine.
		l, err := a.Store.GetNPCPoolListing(uid, body.ListingID-npcIDOffset)
		if err != nil {
			return err
		}
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			if err := a.Store.DeleteNPCPoolStoneTx(tx, l.ID, uid); err != nil {
				return err
			}
			b, err := store.UpdateChipsTx(tx, uid, -l.AskPrice)
			if err != nil {
				return err
			}
			bal = b
			return a.Store.SetStoneStateTx(tx, l.StoneID, domain.StateOwned, uid)
		}); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "chips": bal, "stone_id": l.StoneID})
		return nil
	}
	l, err := a.Store.GetListing(body.ListingID)
	if err != nil {
		return err
	}
	if l.SellerID == uid {
		return errors.New("不能买自己挂的单")
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if err := a.Store.BuyListingTx(tx, l.ID, uid); err != nil {
			return err
		}
		b, err := store.UpdateChipsTx(tx, uid, -l.AskPrice)
		if err != nil {
			return err
		}
		bal = b
		// anonymous seller still gets paid server-side (95% after commission)
		if _, err = tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`, l.AskPrice-l.AskPrice/20, l.SellerID); err != nil {
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
	// 門檻 5000：籌碼低於 5000 就可以領（2026-09-16 由「歸零」放寬）
	if u.Chips >= domain.ReliefThreshold {
		return fmt.Errorf("籌碼還有 %d（%d 以下才能領救濟）", u.Chips, domain.ReliefThreshold)
	}
	// 每 24 小時最多 3 次（2026-09-17 用戶要求：1 天 3 次）
	since := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	used, err := a.Store.ReliefCountSince(int64(uid), since)
	if err != nil {
		return err
	}
	if used >= domain.ReliefPerDay {
		return fmt.Errorf("24 小時內最多領 %d 次救濟，你已經領了 %d 次喵", domain.ReliefPerDay, used)
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
			if _, err := tx.Exec(`INSERT INTO relief_log(user_id, at) VALUES(?, ?)`, uid, now); err != nil {
				return err
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
		if _, err := tx.Exec(`INSERT INTO relief_log(user_id, at) VALUES(?, ?)`, uid, now); err != nil {
			return err
		}
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
