package api

import (
	"fmt"
	"net/http"
	"strings"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// ---------- inventory ----------

func (a *API) inventoryList(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	stones, err := a.Store.ListStonesByOwner(uid)
	if err != nil {
		return err
	}
	items := []map[string]any{}
	for _, st := range stones {
		card := map[string]any{
			"id": st.ID, "seed": seedStr(st.Seed), "grade": int(st.Grade),
			"price": st.Price, "hint": st.LightHint, "state": string(st.State), "origin": st.Origin,
		}
		if st.Grade == domain.WindowGrade && !st.InkHidden {
			card["window_desc"] = windowDesc(st)
		}
		items = append(items, card)
	}
	writeJSON(w, 200, map[string]any{"stones": items})
	return nil
}

// ---------- helpers ----------

// loadOwnedStone fetches a stone and validates ownership + state.
func (a *API) loadOwnedStone(r *http.Request, uid int) (*domain.Stone, error) {
	var body struct {
		StoneID string `json:"stone_id"`
	}
	if err := readJSON(nil, r, &body); err != nil {
		return nil, err
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil {
		return nil, store.ErrNotFound
	}
	if st.OwnerID != uid || st.State != domain.StateOwned {
		return nil, store.ErrNotOwned
	}
	return st, nil
}

// discover credits collection score for a variety if new.
func (a *API) discover(tx *store.Tx, uid int, v domain.ColorVariety) (int, bool, error) {
	first, err := a.Store.DiscoverTx(tx, uid, v)
	if err != nil {
		return 0, false, err
	}
	if !first {
		return 0, false, nil
	}
	score := v.CollectionScore()
	if _, err := tx.Exec(`UPDATE users SET collection_score = collection_score + ? WHERE id=?`, score, uid); err != nil {
		return 0, false, err
	}
	return score, true, nil
}

// ---------- 切 cut ----------

func (a *API) cut(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	var body struct {
		StoneID      string `json:"stone_id"`
		DoubleCoupon bool   `json:"double_coupon"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	st, err := a.Store.GetStone(body.StoneID)
	if err != nil {
		return store.ErrNotFound
	}
	if st.OwnerID != uid || st.State != domain.StateOwned {
		return store.ErrNotOwned
	}

	useCoupon := false
	var payout int
	var firstScore int
	var isNew bool
	var bal int
	var titleAwarded string
	var streakAfter int

	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if body.DoubleCoupon {
			ok, err := a.Store.ConsumeItemTx(tx, uid, "double_coupon")
			if err != nil {
				return err
			}
			useCoupon = ok // if not held, proceed without coupon
		}
		payout = domain.CutReveal(st, useCoupon)

		// discovery + collection score
		if st.Variety.IsExotic() || st.Quality >= domain.Icy {
			s, isNew2, err := a.discover(tx, uid, st.Variety)
			if err != nil {
				return err
			}
			firstScore, isNew = s, isNew2
		}

		// streak bookkeeping
		if st.Quality == domain.Brick {
			_, err := tx.Exec(`UPDATE users SET streak_brick = streak_brick + 1 WHERE id=?`, uid)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(`UPDATE users SET streak_brick = 0, daily_win_streak = daily_win_streak + 1 WHERE id=?`, uid)
			if err != nil {
				return err
			}
		}
		if err := tx.QueryRow(`SELECT streak_brick FROM users WHERE id=?`, uid).Scan(&streakAfter); err != nil {
			return err
		}
		// 一刀披麻布 egg: 4th cut after 3 bricks
		if streakAfter >= 4 {
			_, err := tx.Exec(`UPDATE users SET title='賭徒之魂' WHERE id=?`, uid)
			if err != nil {
				return err
			}
			titleAwarded = "賭徒之魂"
		}
		// 黃金瞳 egg: 5 wins in one day
		if strings.EqualFold(st.Quality.Name(), "玻璃种") || payout >= st.Price*5 {
			ws := 0
			_ = tx.QueryRow(`SELECT daily_win_streak FROM users WHERE id=?`, uid).Scan(&ws)
			if ws >= 5 {
				_, err := tx.Exec(`UPDATE users SET title='黃金瞳' WHERE id=?`, uid)
				if err != nil {
					return err
				}
				titleAwarded = "黃金瞳"
			}
		}

		// payout
		if payout > 0 {
			b, err := store.UpdateChipsTx(tx, uid, payout)
			if err != nil {
				return err
			}
			bal = b
		}
		if err := a.Store.SetStoneStateTx(tx, st.ID, domain.StateUsed, uid); err != nil {
			return err
		}

		// hall of fame for imperial green or 10× payout
		if st.Variety == domain.ImperialGreen || payout >= st.Price*10 {
			name, err := store.UsernameTx(tx, uid)
			if err != nil {
				return err
			}
			if err := a.Store.AddHallOfFameTx(tx, uid, name, st, payout); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	writeJSON(w, 200, map[string]any{
		"payout": payout, "chips": bal,
		"quality": st.Quality.Name(), "variety": st.Variety.Name(),
		"cracks": st.CrackCells, "deep_crack": st.CracksDeep,
		"first_discovery": isNew, "collection_gain": firstScore,
		"title_awarded": titleAwarded, "egg": st.Egg,
		"multiplier": fmt.Sprintf("%.2f", float64(payout)/float64(st.Price)),
	})
	return nil
}
