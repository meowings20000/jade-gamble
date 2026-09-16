package api

import (
	"errors"
	"net/http"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// ---------- 磨 polish (crash) ----------

func (a *API) polishStart(w http.ResponseWriter, r *http.Request) error {
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
	// resume or create
	prog, err := a.Store.GetPolishProgress(st.ID)
	if err != nil {
		return err
	}
	if prog == nil {
		// insurance coupon reduces break chance by 8pp? no — insurance refunds.
		breakMod := 0.0
		has, _ := a.Store.HasActiveBuff(uid, "polish_touch", nowUTC())
		if has {
			breakMod = 0.08
		}
		if err := a.Store.SavePolishProgress(st.ID, uid, 0, true, breakMod); err != nil {
			return err
		}
		prog, _ = a.Store.GetPolishProgress(st.ID)
	}
	writeJSON(w, 200, map[string]any{
		"stone_id": st.ID, "seed": seedStr(st.Seed),
		"stage": prog.Stage, "multiplier": domain.DefaultPolish.Multipliers[prog.Stage],
		"ladder": domain.DefaultPolish.Multipliers, "alive": prog.Alive,
	})
	return nil
}

func (a *API) polishAdvance(w http.ResponseWriter, r *http.Request) error {
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
	prog, err := a.Store.GetPolishProgress(st.ID)
	if err != nil || prog == nil {
		return errors.New("polish session not found")
	}
	if !prog.Alive {
		return errors.New("这颗石头已经磨崩了")
	}
	if prog.Stage >= len(domain.DefaultPolish.Multipliers)-1 {
		return errors.New("已到顶")
	}
	ps := &domain.PolishState{Stage: prog.Stage, Alive: prog.Alive}
	alive, brokeAt := ps.Advance(domain.RandSource, prog.BreakMod)
	if err := a.Store.SavePolishProgress(st.ID, uid, ps.Stage, ps.Alive, prog.BreakMod); err != nil {
		return err
	}
	resp := map[string]any{
		"stage":      ps.Stage,
		"multiplier": domain.DefaultPolish.Multipliers[ps.Stage],
		"alive":      alive,
	}
	if !alive {
		// 磨崩: stone destroyed. Insurance refunds 50% of base.
		refund := 0
		var bal int
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			used, err := a.Store.ConsumeItemTx(tx, uid, "insurance")
			if err != nil {
				return err
			}
			if used {
				refund = st.BaseValue() / 2
				if refund > 0 {
					b, err := store.UpdateChipsTx(tx, uid, refund)
					if err != nil {
						return err
					}
					bal = b
				}
			}
			return a.Store.SetStoneStateTx(tx, st.ID, domain.StateUsed, uid)
		}); err != nil {
			return err
		}
		resp["broke_at"] = brokeAt
		resp["insurance_refund"] = refund
		resp["chips"] = bal
	}
	writeJSON(w, 200, resp)
	return nil
}

func (a *API) polishCash(w http.ResponseWriter, r *http.Request) error {
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
	prog, err := a.Store.GetPolishProgress(st.ID)
	if err != nil || prog == nil || !prog.Alive {
		return errors.New("polish session invalid")
	}
	ps := &domain.PolishState{Stage: prog.Stage, Alive: prog.Alive}
	payout := ps.CashPayout(st.BaseValue())
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
		"payout": payout, "chips": bal, "stage": prog.Stage,
		"quality": st.Quality.Name(), "variety": st.Variety.Name(),
		"first_discovery": isNew, "collection_gain": firstScore,
	})
	return nil
}

// ---------- 套型 setting ----------

func (a *API) setting(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID string `json:"stone_id"`
		Mold    int    `json:"mold"`
		Anchor  int    `json:"anchor"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Mold < 0 || body.Mold > 2 || body.Anchor < 0 || body.Anchor >= 96 {
		return errors.New("mold/anchor out of range")
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil || st.OwnerID != uid || st.State != domain.StateUsed {
		return store.ErrNotOwned
	}
	if st.Origin == "classic" {
		return errors.New("传统石不能套型")
	}
	// setting requires the stone to have been fully revealed (cut or scratched)
	// — StateUsed is only reachable via those paths, and polish breaks don't
	// allow setting. Track via scratch/polish progress rows.
	prog, _ := a.Store.GetPolishProgress(st.ID)
	if prog != nil {
		return errors.New("磨崩的石頭不能套型")
	}
	mold := domain.Mold(body.Mold)
	res := domain.ComputeSetting(st, mold, body.Anchor, st.BaseValue())
	var bal int
	var firstScore int
	var isNew bool
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		// replace base value with setting payout: net (payout - base) delta
		delta := res.Payout - st.BaseValue()
		if delta != 0 {
			b, err := store.UpdateChipsTx(tx, uid, delta)
			if err != nil {
				return err
			}
			bal = b
		}
		_ = firstScore
		_ = isNew
		return a.Store.SetStoneStateTx(tx, st.ID, "set", uid)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"payout": res.Payout, "chips": bal, "mult": res.Mult,
		"cracks_hit": res.CracksHit, "deep_hit": res.DeepHit,
	})
	return nil
}
