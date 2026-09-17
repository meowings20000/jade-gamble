package api

import "net/http"

// bankDebug: GET /api/bank/_debug —— 暫時的診斷端點（只看得到自己的資料）。
func (a *API) bankDebug(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	loans, err := a.Store.DebugLoanRows(uid)
	if err != nil {
		return err
	}
	chat, err := a.Store.DebugChatAll()
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "uid": uid, "loans": loans, "recent_chat": chat})
	return nil
}
