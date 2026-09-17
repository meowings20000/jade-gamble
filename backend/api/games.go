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
		Force   int    `json:"force"` // 1 輕磨 / 2 正磨 / 3 重磨（resume 時可省略）
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
		if body.Force < domain.PolishForceLight || body.Force > domain.PolishForceHeavy {
			return errors.New("先選磨石力度（輕磨／正磨／重磨）")
		}
		// 開磨損耗：皮殼一磨掉就不值原石價
		breakMod := 0.0
		has, _ := a.Store.HasActiveBuff(uid, "polish_touch", nowUTC())
		if has {
			breakMod = 0.08
		}
		if err := a.Store.SavePolishProgress(st.ID, uid, 0, true, breakMod, body.Force); err != nil {
			return err
		}
		prog, _ = a.Store.GetPolishProgress(st.ID)
	} else if body.Force >= domain.PolishForceLight && body.Force <= domain.PolishForceHeavy &&
		body.Force != prog.Force {
		// 還沒磨掉任何一層（stage 0）＝只是選了力度還沒動工，可以反悔重選；
		// 磨過一層之後就不能改（不然可以看手感再換，等於免費情報）。
		if prog.Stage > 0 {
			return errors.New("已經磨過一層了，力度不能再改")
		}
		if err := a.Store.SavePolishProgress(st.ID, uid, 0, true, prog.BreakMod, body.Force); err != nil {
			return err
		}
		prog, _ = a.Store.GetPolishProgress(st.ID)
	}
	writeJSON(w, 200, map[string]any{
		"stone_id": st.ID, "seed": seedStr(st.Seed),
		"stage": prog.Stage, "multiplier": domain.PolishMultiplier(st, prog.Stage),
		"ladder": map[string]any{
			"start": domain.PolishStartMult, "gain": domain.PolishStepGain,
			"base_value": st.BaseValue(), "cash_value": int(float64(st.BaseValue()) * domain.PolishMultiplier(st, prog.Stage)),
			"top": domain.PolishCeilingFor(st), "max_stage": domain.PolishMaxStage,
		},
		"base_value": st.BaseValue(), "cash_value": int(float64(st.BaseValue()) * domain.PolishMultiplier(st, prog.Stage)), "stone_price": st.Price,
		"alive": prog.Alive, "force": prog.Force, "force_name": domain.PolishForceName(prog.Force),
		"break_prob": func() float64 {
			pr := domain.PolishBreakProb(st, prog.Force, prog.BreakMod)
			if prog.Stage == 0 && pr > domain.PolishFirstLayerCap {
				pr = domain.PolishFirstLayerCap
			}
			return pr
		}(),
		"feel":   domain.PolishFeel(st, prog.Force),
		"at_top": prog.Stage >= domain.PolishMaxStage,
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
	if prog.Stage >= domain.PolishMaxStage {
		return errors.New("已到顶")
	}
	ps := &domain.PolishState{Stage: prog.Stage, Force: prog.Force, Alive: prog.Alive}
	alive, brokeAt := ps.Advance(st, domain.RandSource, prog.BreakMod)
	if err := a.Store.SavePolishProgress(st.ID, uid, ps.Stage, ps.Alive, prog.BreakMod, prog.Force); err != nil {
		return err
	}
	resp := map[string]any{
		"stage":       ps.Stage,
		"multiplier":  domain.PolishMultiplier(st, ps.Stage),
		"base_value":  st.BaseValue(),
		"cash_value":  ps.CashPayout(st, st.BaseValue()),
		"stone_price": st.Price,
		"alive":       alive,
		"broke_at":    brokeAt,
		"feel":        domain.PolishFeel(st, ps.Force),
		"break_prob":  domain.PolishBreakProb(st, ps.Force, prog.BreakMod),
		"quality":     st.Quality.Name(),
		"variety":     st.Variety.Name(),
		"force":       ps.Force,
		"force_name":  domain.PolishForceName(ps.Force),
		"at_top":      ps.Stage >= domain.PolishMaxStage,
	}
	if !alive {
		// 磨崩：石頭報廢，但救回當前倍率的 35%（2026-09-17：避免一次失手就血本無歸）
		salvage := domain.PolishBrokenPayout(st, prog.Stage, st.BaseValue())
		refund := 0
		var bal int
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			if salvage > 0 {
				b, err := store.UpdateChipsTx(tx, uid, salvage)
				if err != nil {
					return err
				}
				bal = b
			}
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
		resp["salvage"] = salvage
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
	ps := &domain.PolishState{Stage: prog.Stage, Force: prog.Force, Alive: prog.Alive}
	payout := ps.CashPayout(st, st.BaseValue())
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
		if err := a.Store.LogStoneTx(tx, uid, st.ID, "polish", int(st.Grade),
			st.Quality.Name(), st.Variety.Name(), st.Price, payout); err != nil {
			return err
		}
		return a.Store.SetStoneStateTx(tx, st.ID, domain.StateUsed, uid)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"payout": payout, "chips": bal, "stage": prog.Stage,
		"polish": true, "stone_price": st.Price, "net": payout - st.Price,
		"multiplier":      domain.PolishMultiplier(st, prog.Stage),
		"quality":         st.Quality.Name(),
		"variety":         st.Variety.Name(),
		"first_discovery": isNew, "collection_gain": firstScore,
		"force": prog.Force, "force_name": domain.PolishForceName(prog.Force),
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
