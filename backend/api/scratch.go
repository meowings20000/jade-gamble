package api

import (
	"errors"
	"net/http"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// perCellFor: 每格價值（與 domain.perCellOf 同規則，至少 1 籌碼）。
func perCellFor(st *domain.Stone) int {
	v := st.BaseValue() / domain.ScratchCells
	if v < 1 {
		v = 1
	}
	return v
}

// ---------- 刮 scratch (brush UX; server-authoritative layout) ----------

// scratchStart opens (or resumes) a scratch session.
func (a *API) scratchStart(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	st, err := a.loadOwnedStone(r, uid)
	if err != nil {
		return err
	}
	prog, err := a.Store.GetScratchProgress(st.ID)
	if err != nil {
		return err
	}
	if prog == nil {
		// crypto-random layout, persisted server-side (never seed-derived)
		ss := domain.NewScratch(st, domain.RandSource)
		layout := store.ScratchLayout{CrackAt: ss.CrackAt, DeepAt: ss.DeepAt}
		if err := a.Store.SaveScratchProgressFull(st.ID, uid, ss, layout); err != nil {
			return err
		}
		prog, _ = a.Store.GetScratchProgress(st.ID)
	}
	writeJSON(w, 200, map[string]any{
		"stone_id": st.ID, "seed": seedStr(st.Seed), "grade": int(st.Grade),
		"revealed": prog.RevealedKinds(), "accumulated": prog.Accumulated,
		"done": prog.Done, "hint": hintFor(a, st),
	})
	return nil
}

// scratchReveal opens one cell (the brush reached ~50% coverage).
func (a *API) scratchReveal(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID string `json:"stone_id"`
		Cell    int    `json:"cell"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Cell < 0 || body.Cell >= domain.ScratchCells {
		return errors.New("cell out of range")
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil || st.OwnerID != uid || st.State != domain.StateOwned {
		return store.ErrNotOwned
	}
	prog, err := a.Store.GetScratchProgress(st.ID)
	if err != nil || prog == nil {
		return errors.New("scratch session not found")
	}
	if prog.Revealed[body.Cell] {
		return errors.New("cell already revealed")
	}
	kind := prog.Layout.KindAt(body.Cell)

	// 由「開格順序」重算累積值（順序有意義：裂紋是乘法）。
	ss := &domain.ScratchState{
		Revealed: prog.Revealed,
		CrackAt:  prog.Layout.CrackAt,
		DeepAt:   prog.Layout.DeepAt,
		Order:    append(append([]int{}, prog.Order...), body.Cell),
		PerCell:  perCellFor(st),
	}
	ss.Recompute()

	if err := a.Store.SaveScratchProgressFull(st.ID, uid, ss, prog.Layout); err != nil {
		return err
	}
	resp := map[string]any{
		"cell": body.Cell, "kind": kind,
		"accumulated": ss.Accumulated, "done": ss.Done,
	}
	if ss.Done {
		var payout int
		var bal int
		var firstScore int
		var isNew bool
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			payout = ss.Accumulated
			if payout > 0 {
				b, err := store.UpdateChipsTx(tx, uid, payout)
				if err != nil {
					return err
				}
				bal = b
			}
			if st.Variety.IsExotic() || st.Quality >= domain.Icy {
				s, isNew2, err := a.discover(tx, uid, st.Variety)
				if err != nil {
					return err
				}
				firstScore, isNew = s, isNew2
			}
			if err := a.Store.LogStoneTx(tx, uid, st.ID, "scratch", int(st.Grade),
				st.Quality.Name(), st.Variety.Name(), st.Price, payout); err != nil {
				return err
			}
			return a.Store.SetStoneStateTx(tx, st.ID, domain.StateUsed, uid)
		}); err != nil {
			return err
		}
		resp["payout"] = payout
		resp["chips"] = bal
		resp["quality"] = st.Quality.Name()
		resp["variety"] = st.Variety.Name()
		resp["first_discovery"] = isNew
		resp["collection_gain"] = firstScore
	}
	writeJSON(w, 200, resp)
	return nil
}

func (a *API) scratchHint(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID string `json:"stone_id"`
		Cell    int    `json:"cell"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Cell < 0 || body.Cell >= domain.ScratchCells {
		return errors.New("cell out of range")
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil || st.OwnerID != uid || st.State != domain.StateOwned {
		return store.ErrNotOwned
	}
	prog, err := a.Store.GetScratchProgress(st.ID)
	if err != nil || prog == nil {
		return errors.New("scratch session not found")
	}
	truth := prog.Layout.CrackAt[body.Cell]
	if domain.RandSource.Float64() < 0.30 {
		truth = !truth
	}
	hint := "看起来干净，可以下刀。"
	if truth {
		hint = "这条线走色发闷，风险大。"
	}
	writeJSON(w, 200, map[string]any{"hint": hint})
	return nil
}

func (a *API) scratchSell(w http.ResponseWriter, r *http.Request) error {
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
	if err != nil || st.OwnerID != uid || st.State != domain.StateOwned {
		return store.ErrNotOwned
	}
	prog, err := a.Store.GetScratchProgress(st.ID)
	if err != nil || prog == nil {
		return errors.New("scratch session not found")
	}
	if prog.Done {
		return errors.New("已经刮完，直接领奖")
	}
	payout := prog.SellNowFee()
	var bal int
	var firstScore int
	var isNew bool
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, payout)
		if err != nil {
			return err
		}
		bal = b
		if st.Variety.IsExotic() || st.Quality >= domain.Icy {
			s, isNew2, err := a.discover(tx, uid, st.Variety)
			if err != nil {
				return err
			}
			firstScore, isNew = s, isNew2
		}
		return a.Store.SetStoneStateTx(tx, st.ID, domain.StateUsed, uid)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"payout": payout, "chips": bal,
		"quality": st.Quality.Name(), "variety": st.Variety.Name(),
		"first_discovery": isNew, "collection_gain": firstScore,
	})
	return nil
}
