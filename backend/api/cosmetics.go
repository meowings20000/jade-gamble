package api

// 裝飾品裝備系統（2026-10-06）：頭像框等 cosmetic 的更換入口
//
// GET  /api/cosmetics        我擁有的裝飾品 + 目前裝備
// POST /api/cosmetics/equip  {key: "frame_gold" | ""} 裝備/卸下（""= 自動）

import (
	"errors"
	"net/http"
	"strings"
)

var frameKeys = []string{"frame_rainbow", "frame_ink", "frame_imperial", "frame_violet", "frame_gold", "frame_cat"}

// cosmeticsList: GET /api/cosmetics
func (a *API) cosmeticsList(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	owned := []map[string]any{}
	for _, k := range frameKeys {
		if a.Store.OwnsItem(uid, k) {
			owned = append(owned, map[string]any{"key": k})
		}
	}
	writeJSON(w, 200, map[string]any{
		"ok": true,
		"owned": owned,
		"equipped": a.Store.EquippedFrame(uid),
	})
	return nil
}

// cosmeticsEquip: POST /api/cosmetics/equip {key}
func (a *API) cosmeticsEquip(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		// 卸下 = 回自動（最貴）
		if err := a.Store.SetEquippedFrame(uid, ""); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "equipped": a.Store.EquippedFrame(uid)})
		return nil
	}
	valid := false
	for _, k := range frameKeys {
		if k == key {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("未知的頭像框")
	}
	if !a.Store.OwnsItem(uid, key) {
		return errors.New("還沒兌換這個頭像框")
	}
	if err := a.Store.SetEquippedFrame(uid, key); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "equipped": key})
	return nil
}