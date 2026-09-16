package api

import (
	"errors"
	"net/http"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// ---------- Y佬模式 (YBoss): pure stake gambling ----------

// ybossBet: {stake, choice:"cut"|"polish"}. Cut resolves immediately;
// polish opens a session at rung 0 (one live session per user).
func (a *API) ybossBet(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Stake  int    `json:"stake"`
		Choice string `json:"choice"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Stake < domain.YBossMinStake || body.Stake > domain.YBossMaxStake {
		return errors.New("賭資需在 100 ~ 1,000,000 之間")
	}
	if body.Choice != "cut" && body.Choice != "polish" {
		return domain.ErrYBossChoice
	}

	if body.Choice == "polish" {
		open, err := a.Store.HasYBossSessionTx(uid)
		if err != nil {
			return err
		}
		if open {
			return errors.New("還有一場磨石在進行中")
		}
	}

	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, -body.Stake)
		if err != nil {
			return err
		}
		bal = b
		if body.Choice == "polish" {
			return a.Store.CreateYBossSessionTx(tx, uid, body.Stake)
		}
		return nil
	}); err != nil {
		return err
	}

	if body.Choice == "cut" {
		mult, label := domain.YBossCut(domain.RandSource)
		payout := int(float64(body.Stake) * mult)
		if payout > 0 {
			if err := a.Store.WithTx(func(tx *store.Tx) error {
				b, err := store.UpdateChipsTx(tx, uid, payout)
				bal = b
				return err
			}); err != nil {
				return err
			}
		}
		writeJSON(w, 200, map[string]any{
			"choice": "cut", "stake": body.Stake,
			"mult": mult, "label": label,
			"payout": payout, "chips": bal,
		})
		return nil
	}

	writeJSON(w, 200, map[string]any{
		"choice": "polish", "stake": body.Stake,
		"rung": 0, "mult": 1.0, "label": domain.YBossPolishLadder[0].Label,
		"chips": bal, "ladder": ybossLadderView(),
	})
	return nil
}

func ybossLadderView() []map[string]any {
	out := []map[string]any{}
	for _, rg := range domain.YBossPolishLadder {
		out = append(out, map[string]any{"mult": rg.Mult, "label": rg.Label})
	}
	return out
}

// ybossPolish: {advance:true} → roll one rung; {advance:false} → cash out.
func (a *API) ybossPolish(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		Advance bool `json:"advance"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	sess, err := a.Store.GetYBossSession(uid)
	if err != nil {
		return err
	}
	if sess == nil || sess.Finished {
		return errors.New("沒有進行中的Y佬磨石")
	}

	if !body.Advance { // cash out at current rung
		payout := domain.YBossPolishPayout(sess.Stake, sess.Rung)
		var bal int
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			if payout > 0 {
				b, err := store.UpdateChipsTx(tx, uid, payout)
				if err != nil {
					return err
				}
				bal = b
			}
			return a.Store.FinishYBossSessionTx(tx, sess.ID)
		}); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{
			"action": "cash", "payout": payout, "rung": sess.Rung,
			"mult":  domain.YBossPolishLadder[sess.Rung].Mult,
			"label": domain.YBossPolishLadder[sess.Rung].Label,
			"chips": bal,
		})
		return nil
	}

	if sess.Rung >= len(domain.YBossPolishLadder)-1 {
		return errors.New("已在頂層")
	}
	newRung, alive := domain.YBossPolishAdvance(domain.RandSource, sess.Rung)
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if alive {
			return a.Store.SetYBossRungTx(tx, sess.ID, newRung)
		}
		return a.Store.FinishYBossSessionTx(tx, sess.ID)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"action": "advance", "alive": alive, "rung": newRung,
		"mult":  domain.YBossPolishLadder[newRung].Mult,
		"label": domain.YBossPolishLadder[newRung].Label,
	})
	return nil
}
