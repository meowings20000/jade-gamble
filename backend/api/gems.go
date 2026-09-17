package api

import (
	"net/http"

	"jade-gamble/backend/domain"
)

// gemCollection: GET /api/gems —— 寶石圖鑒：全部彩蛋寶石 + 我切到過幾次／首次時間。
func (a *API) gemCollection(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	owned, err := a.Store.GemCollection(uid)
	if err != nil {
		return err
	}
	odds := domain.GemOdds()
	list := make([]map[string]any, 0, len(odds))
	got, total := 0, 0
	for _, g := range odds {
		k, _ := g["key"].(string)
		o := owned[k]
		if o.Count > 0 {
			got++
		}
		total += o.Count
		list = append(list, map[string]any{
			"key":      k,
			"name":     g["name"],
			"mult":     g["mult"],
			"prob":     g["prob"],
			"overall":  g["overall"],
			"count":    o.Count,
			"first_at": o.FirstAt,
		})
	}
	writeJSON(w, 200, map[string]any{
		"ok": true, "chance": domain.GemChance, "gems": list,
		"kinds_owned": got, "kinds_total": len(list), "count_total": total,
	})
	return nil
}
