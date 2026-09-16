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
			"price": st.Price, "hint": hintFor(a, st), "state": string(st.State), "origin": st.Origin,
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
	var cutGem *domain.Gem

	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if body.DoubleCoupon {
			ok, err := a.Store.ConsumeItemTx(tx, uid, "double_coupon")
			if err != nil {
				return err
			}
			useCoupon = ok // if not held, proceed without coupon
		}
		var gem *domain.Gem
		payout, gem = domain.CutRevealWithGem(st, useCoupon, domain.RandSource)
		cutGem = gem

		// discovery + collection score
		if st.Variety.IsExotic() || st.Quality >= domain.Icy {
			s, isNew2, err := a.discover(tx, uid, st.Variety)
			if err != nil {
				return err
			}
			firstScore, isNew = s, isNew2
		}

		// streak bookkeeping
		// 跨日先把當日連勝歸零（以前 daily_win_streak 只加不減、也不看輸贏，
		// 結果活躍帳號人人一個「黃金瞳」）。
		var winDate string
		_ = tx.QueryRow(`SELECT daily_win_date FROM users WHERE id=?`, uid).Scan(&winDate)
		if winDate != today() {
			if _, err := tx.Exec(`UPDATE users SET daily_win_streak = 0, daily_win_date = ? WHERE id=?`, today(), uid); err != nil {
				return err
			}
		}
		if st.Quality == domain.Brick {
			if _, err := tx.Exec(`UPDATE users SET streak_brick = streak_brick + 1 WHERE id=?`, uid); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`UPDATE users SET streak_brick = 0 WHERE id=?`, uid); err != nil {
				return err
			}
		}
		// 只有「真的賺錢」才算連勝，輸一刀就歸零
		if payout >= st.Price {
			if _, err := tx.Exec(`UPDATE users SET daily_win_streak = daily_win_streak + 1 WHERE id=?`, uid); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`UPDATE users SET daily_win_streak = 0 WHERE id=?`, uid); err != nil {
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
		// 黃金瞳：一刀切出 5 倍（或玻璃種）「而且」當日已連贏 5 刀——稀有成就。
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
		// 處理紀錄：玩家看得到自己切了什麼、賺賠多少
		qName, vName := st.Quality.Name(), st.Variety.Name()
		if cutGem != nil {
			qName, vName = cutGem.Name, "彩蛋"
		}
		if err := a.Store.LogStoneTx(tx, uid, st.ID, "cut", int(st.Grade),
			qName, vName, st.Price, payout); err != nil {
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
		"title_awarded": titleAwarded, "egg": gemEggFlag(cutGem, st.Egg),
		"gem": gemKey(cutGem), "gem_name": gemName(cutGem),
		"multiplier": fmt.Sprintf("%.2f", float64(payout)/float64(st.Price)),
	})
	return nil
}

// hintFor: 沒打過燈只給肉眼可見的描述（蒙頭料沒有），打過燈才給報告。
func hintFor(a *API, st *domain.Stone) string {
	if a.Store.IsLit(st.ID) {
		return st.LightHint
	}
	return domain.DescribeFree(st)
}

// gemKey / gemName / gemEggFlag: 彩蛋欄位（nil 時回空字串，前端據此決定要不要畫寶石）。
func gemKey(g *domain.Gem) string {
	if g == nil {
		return ""
	}
	return g.Key
}

func gemName(g *domain.Gem) string {
	if g == nil {
		return ""
	}
	return g.Name
}

func gemEggFlag(g *domain.Gem, stoneEgg string) string {
	if g != nil {
		return "gem"
	}
	return stoneEgg
}
