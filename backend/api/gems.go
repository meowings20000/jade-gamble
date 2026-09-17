package api

import (
	"net/http"

	"jade-gamble/backend/domain"
)

// gemCollection: GET /api/gems —— 寶石圖鑒：全部彩蛋寶石 + 我切到過幾次。
func (a *API) gemCollection(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	odds := domain.GemOdds()
	names := make([]string, 0, len(odds))
	for _, g := range odds {
		names = append(names, g["name"].(string))
	}
	counts, err := a.Store.GemCounts(int64(uid), names)
	if err != nil {
		return err
	}
	list := make([]map[string]any, 0, len(odds))
	for _, g := range odds {
		nm, _ := g["name"].(string)
		list = append(list, map[string]any{
			"key":      g["key"],
			"name":     nm,
			"mult":     g["mult"],
			"prob":     g["prob"],
			"overall":  g["overall"],
			"count":    counts[nm],
			"first_at": "",
		})
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chance": domain.GemChance, "gems": list})
	return nil
}
