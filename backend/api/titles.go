package api

import (
	"errors"
	"net/http"

	"jade-gamble/backend/domain"
)

// titles: GET /api/titles —— 我的稱號（哪些解鎖、現在裝哪個）。
func (a *API) titles(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	ts, err := a.Store.TitleStats(uid)
	if err != nil {
		return err
	}
	unlocked := map[string]bool{}
	for _, k := range domain.UnlockedTitles(ts) {
		unlocked[k] = true
	}
	list := []map[string]any{}
	for _, t := range domain.Titles {
		list = append(list, domain.TitleView(t, unlocked[t.Key], u.Title == t.Name))
	}
	writeJSON(w, 200, map[string]any{
		"titles": list, "equipped": u.Title,
		"stats": map[string]any{
			"cuts": ts.Cuts, "scratches": ts.Scratches, "polishes": ts.Polishes,
			"best_mult": ts.BestMult, "gems": ts.Gems, "diamonds": ts.Diamonds,
			"varieties": ts.Varieties, "collection": ts.Collection,
			"chips": ts.Chips, "brick_streak": ts.BrickStreak,
		},
	})
	return nil
}

// equipTitle: POST /api/titles/equip {key} —— 裝上稱號（key 空 = 不顯示）。
func (a *API) equipTitle(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	if body.Key == "" {
		if err := a.Store.EquipTitle(uid, ""); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "message": "已卸下稱號"})
		return nil
	}
	t, ok := domain.TitleByKey(body.Key)
	if !ok {
		return errors.New("沒有這個稱號")
	}
	ts, err := a.Store.TitleStats(uid)
	if err != nil {
		return err
	}
	allowed := false
	for _, k := range domain.UnlockedTitles(ts) {
		if k == t.Key {
			allowed = true
		}
	}
	if !allowed {
		return errors.New("還沒解鎖這個稱號")
	}
	if err := a.Store.EquipTitle(uid, t.Name); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "已裝上【" + t.Name + "】"})
	return nil
}
