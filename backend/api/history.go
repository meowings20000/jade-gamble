package api

import "net/http"

// history: GET /api/history —— 我處理過的石頭紀錄 + 總帳。
func (a *API) history(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	limit := atoiDefault(r.URL.Query().Get("limit"), 60)
	entries, err := a.Store.ListHistory(uid, limit)
	if err != nil {
		return err
	}
	stats, err := a.Store.HistoryStats(uid)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"entries": entries, "stats": stats})
	return nil
}

func atoiDefault(s string, def int) int {
	n := 0
	if s == "" {
		return def
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}
