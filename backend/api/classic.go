package api

import (
	"errors"
	"net/http"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// classicBet: traditional mode (传统模式). The player stakes chips, the
// server rolls one blind stone priced at the stake. No hints, no resale,
// no setting — only cut / polish / scratch afterwards.
func (a *API) classicBet(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Stake int `json:"stake"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Stake < domain.MinStake || body.Stake > domain.MaxStake {
		return domain.ErrInvalidStake
	}
	var stoneID string
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		// check no unfinished classic stone first (one at a time)
		pending, err := a.Store.HasClassicPendingTx(tx, uid)
		if err != nil {
			return err
		}
		if pending {
			return errors.New("你还有一颗传统石在处理中（先切/磨/刮完）")
		}
		b, err := store.UpdateChipsTx(tx, uid, -body.Stake)
		if err != nil {
			return err
		}
		bal = b
		st := domain.GenerateClassicStone(body.Stake, domain.RandSource)
		st.OwnerID = uid
		st.State = domain.StateOwned
		if err := a.Store.SaveStoneTx(tx, st); err != nil {
			return err
		}
		stoneID = st.ID
		return nil
	}); err != nil {
		return err
	}
	// public view: seed + grade band + stake only (no hint!)
	writeJSON(w, 200, map[string]any{
		"stone_id": stoneID,
		"stake":    body.Stake,
		"band":     domain.StakeBand(body.Stake).Name(),
		"seed":     0, // seed fetched via inventory
		"chips":    bal,
	})
	return nil
}
