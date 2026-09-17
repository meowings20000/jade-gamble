package api

import (
	"fmt"
	"net/http"
	"time"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

func today() string { return time.Now().Format("2006-01-02") }

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

// shopView returns the user's three shelves with visible stone info.
// The client NEVER receives quality/variety/cracks for shelf stones.
func (a *API) shopView(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	newDay, err := a.Store.EnsureShelf(uid, today())
	if err != nil {
		return err
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	out := map[string]any{"chips": u.Chips, "grades": []map[string]any{}}
	grades := out["grades"].([]map[string]any)
	for g := domain.KiloGrade; g <= domain.WindowGrade; g++ {
		ids, err := a.Store.ShelfStones(uid, g)
		if err != nil {
			return err
		}
		// 只有跨日（每日免費補貨）才補滿；單純開頁／跳頁不會生新石頭。
		// 以前這裡是「看到空格就補」，等於買走一格、跳個頁就免費多一顆。
		if newDay && len(ids) < domain.ShelfSize(g) {
			for len(ids) < domain.ShelfSize(g) {
				st := domain.GenerateStone(g, domain.RandSource)
				st.OwnerID = uid
				st.State = domain.StateShop
				if err := a.Store.SaveStone(st); err != nil {
					return err
				}
				if err := a.Store.FillShelfSlot(uid, g, st.ID); err != nil {
					return err
				}
				ids = append(ids, st.ID)
			}
		}
		items := []map[string]any{}
		for _, id := range ids {
			st, err := a.Store.GetStone(id)
			if err != nil {
				return err
			}
			items = append(items, stoneCardLit(st, a.Store.IsLit(st.ID)))
		}
		refreshes, _ := a.Store.CountShelfRefreshes(uid, g, today())
		grades = append(grades, map[string]any{
			"grade": int(g), "name": g.Name(), "items": items,
			"next_refresh": domain.RefreshPrice(g, refreshes),
		})
	}
	out["grades"] = grades
	writeJSON(w, 200, out)
	return nil
}

// stoneCard is the public view of an unopened stone.
func stoneCard(st *domain.Stone) map[string]any {
	return stoneCardLit(st, false)
}

// stoneCardLit: 沒打燈只給「肉眼可見的」描述（蒙頭料什麼都沒有），
// 付費打過燈才給報告（蒙頭模糊、表現中等、開窗準確）。
func stoneCardLit(st *domain.Stone, lit bool) map[string]any {
	hint := domain.DescribeFree(st)
	if lit {
		hint = st.LightHint
	}
	return map[string]any{
		"id": st.ID, "seed": seedStr(st.Seed), "grade": int(st.Grade), "price": st.Price,
		"lit": lit, "hint": hint,
	}
}

func windowDesc(st *domain.Stone) string {
	switch st.Quality {
	case domain.Glass:
		return "窗面起荧，玻璃光泽逼人"
	case domain.Icy:
		return "窗面通透，冰感十足"
	case domain.OilGreen:
		return "窗面油亮，色沉而正"
	case domain.Bean:
		return "窗面可见豆状颗粒，色淡"
	default:
		return "窗面干白，颗粒粗糙"
	}
}

// shopRefresh replaces all stones of one grade, charging the escalating fee.
func (a *API) shopRefresh(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Grade int `json:"grade"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	g := domain.ShopGrade(body.Grade)
	if g < domain.KiloGrade || g > domain.WindowGrade {
		return fmt.Errorf("未知档位")
	}
	// shelves rows must exist before we can clear/fill them
	if _, err := a.Store.EnsureShelf(uid, today()); err != nil {
		return err
	}
	refreshes, err := a.Store.CountShelfRefreshes(uid, g, today())
	if err != nil {
		return err
	}
	cost := domain.RefreshPrice(g, refreshes)

	// Single atomic transaction: consume coupon if held, else charge; refill.
	// NOTE: everything inside must be Tx-aware — the pool is MaxOpenConns(1),
	// any non-tx store call inside the closure deadlocks.
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		usedCoupon, err := a.Store.ConsumeItemTx(tx, uid, "free_refresh")
		if err != nil {
			return err
		}
		charge := cost
		if usedCoupon {
			charge = 0
		}
		if charge > 0 {
			if _, err := store.UpdateChipsTx(tx, uid, -charge); err != nil {
				return err
			}
		}
		if err := a.Store.ClearShelfTx(tx, uid, g); err != nil {
			return err
		}
		for i := 0; i < domain.ShelfSize(g); i++ {
			st := domain.GenerateStone(g, domain.RandSource)
			st.OwnerID = uid
			st.State = domain.StateShop
			if err := a.Store.SaveStoneTx(tx, st); err != nil {
				return err
			}
			if err := a.Store.FillShelfSlotTx(tx, uid, g, st.ID); err != nil {
				return err
			}
		}
		// 用券也要累加當日次數：否則拿著券就能永遠用基準價刷新（價格不會漲）
		return a.Store.IncRefreshTx(tx, uid, g, today())
	}); err != nil {
		return err
	}
	return a.shopView(w, r)
}

// shopBuy purchases a shelf stone.
func (a *API) shopBuy(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID string `json:"stone_id"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil {
		return store.ErrNotFound
	}
	g := st.Grade
	var newBal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		// remove from shelf atomically
		took, err := a.Store.TakeShelfStoneTx(tx, uid, g, st.ID)
		if err != nil {
			return err
		}
		if !took {
			return fmt.Errorf("这颗石头已被人买走")
		}
		bal, err := store.UpdateChipsTx(tx, uid, -st.Price)
		if err != nil {
			return err
		}
		if err := a.Store.SetStoneStateTx(tx, st.ID, domain.StateOwned, uid); err != nil {
			return err
		}
		newBal = bal
		return nil
	}); err != nil {
		return err
	}
	// golden eye buff: reveal an extra true feature tag on feature stones
	writeJSON(w, 200, map[string]any{"ok": true, "chips": newBal, "stone_id": st.ID})
	return nil
}

// shopLightReport buys a flashlight report for a shelf stone (5% of price).
func (a *API) shopLightReport(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID string `json:"stone_id"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil {
		return store.ErrNotFound
	}
	// 公斤料也能打燈：蒙頭料打燈只有模糊描述（見 buildBlindHint）。
	cost := st.Price / 20 // 5%
	hasMaster, _ := a.Store.HasActiveBuff(uid, "light_master", nowUTC())
	if hasMaster {
		cost /= 2
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, -cost)
		bal = b
		return err
	}); err != nil {
		return err
	}
	if err := a.Store.MarkLit(st.ID, uid); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"report": st.LightHint, "chips": bal, "cost": cost})
	return nil
}
