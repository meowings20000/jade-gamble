package api

import (
	"encoding/json"
	"net/http"
)

// 集中活動 8 種（user 2026-10-06）：admin 開關。
var serverEventKeys = map[string]string{
	"polish_luck":   "切石爽感（彩蛋 ×2）",
	"salvage_30":    "磨崩返還 30%",
	"market_fee_0":  "貨架上架免 5%",
	"checkin_x2":    "簽到雙倍",
	"race_prize":    "排行榜獎金 ×2",
	"special_stone": "限定石頭機率提升",
	"heist_bonus":   "奪寶獎池再加 50%",
	"ai_weekend":    "AI 開市門檻半價",
}

// adminEventList: GET /api/admin/server-events
func (a *API) adminServerEventList(w http.ResponseWriter, r *http.Request) error {
	if _, err := a.requireAdmin(w, r); err != nil {
		return err
	}
	list, err := a.Store.ServerEventList()
	if err != nil {
		return err
	}
	metas := []map[string]any{}
	for k, label := range serverEventKeys {
		active := false
		exp := ""
		for _, se := range list {
			if se.Key == k {
				active = true
				exp = se.ExpiresAt
			}
		}
		metas = append(metas, map[string]any{"key": k, "label": label, "active": active, "expires_at": exp})
	}
	writeJSON(w, 200, map[string]any{"ok": true, "events": metas})
	return nil
}

// adminServerEvent: POST /api/admin/server-event {key, hours|0, off}
func (a *API) adminServerEvent(w http.ResponseWriter, r *http.Request) error {
	adminID, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var body struct {
		Key   string `json:"key"`
		Hours int    `json:"hours"`
		Off   bool   `json:"off"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, "參數不對")
		return nil
	}
	if _, known := serverEventKeys[body.Key]; !known {
		writeErr(w, 400, "未知的活動 key")
		return nil
	}
	if body.Off {
		if err := a.Store.ServerEventDeactivate(body.Key); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "msg": "活動已關閉"})
		return nil
	}
	if err := a.Store.ServerEventActivate(body.Key, body.Hours, adminID); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "msg": "活動已開啟"})
	return nil
}
