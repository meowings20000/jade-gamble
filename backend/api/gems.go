package api

import (
	"net/http"

	"jade-gamble/backend/domain"
)

// gemCollection: GET /api/gems —— 寶石圖鑒（全部 6 種 + 我收集到的）。
func (a *API) gemCollection(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	owned, err := a.Store.GemCollection(uid)
	if err != nil {
		return err
	}
	list := []map[string]any{}
	total := 0
	for _, g := range domain.Gems {
		item := map[string]any{"key": g.Key, "name": g.Name, "mult": g.Mult, "prob": g.Prob, "count": 0, "first_at": ""}
		if o, ok := owned[g.Key]; ok {
			item["count"] = o.Count
			item["first_at"] = o.FirstAt
			total += o.Count
		}
		list = append(list, item)
	}
	writeJSON(w, 200, map[string]any{
		"gems": list, "kinds_total": len(domain.Gems), "kinds_owned": len(owned), "count_total": total,
		"chance": domain.GemChance,
	})
	return nil
}
