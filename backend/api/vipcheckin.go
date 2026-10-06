package api

// 簽到 + 升級賬戶 + 黑名單 API（2026-10-06）
//
// POST /api/checkin          每日簽到（一般 10 萬 / VIP 20 萬）
// GET  /api/checkin          狀態（今天簽了沒、能領多少）
// POST /api/account/upgrade  花 100 萬 → 永久升級
// POST /api/bank/repay-debt  {amount} 還欠款（清零自動開 3 天冷靜期）
// GET  /api/bank/blacklist   黑名單榜

import (
	"fmt"
	"net/http"
	"time"
)

func shortChips(n int) string {
	switch {
	case n >= 100000000:
		return fmt.Sprintf("%.1f億", float64(n)/100000000)
	case n >= 10000:
		return fmt.Sprintf("%d萬", n/10000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// checkinStatus: GET /api/checkin
func (a *API) checkinStatus(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	checked, amount, vip := a.Store.CheckinStatus(uid, time.Now())
	writeJSON(w, 200, map[string]any{
		"ok": true, "checked": checked, "amount": amount, "vip": vip,
	})
	return nil
}

// checkin: POST /api/checkin
func (a *API) checkin(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	chips, amount, err := a.Store.Checkin(uid, time.Now())
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": chips, "amount": amount})
	return nil
}

// accountUpgrade: POST /api/account/upgrade
func (a *API) accountUpgrade(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	chips, err := a.Store.UpgradeAccount(uid, time.Now())
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": chips, "vip": true})
	return nil
}

// bankRepayDebt: POST /api/bank/repay-debt
func (a *API) bankRepayDebt(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Amount int `json:"amount"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	debt, chips, err := a.Store.RepayDebt(uid, body.Amount, time.Now())
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "debt": debt, "chips": chips})
	return nil
}

// bankBlacklist: GET /api/bank/blacklist
func (a *API) bankBlacklist(w http.ResponseWriter, r *http.Request) error {
	board, err := a.Store.BlacklistBoard(20)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "board": board})
	return nil
}

// 玩家名單（轉賬 pulldown 用）：GET /api/players
func (a *API) playersList(w http.ResponseWriter, r *http.Request) error {
	if _, err := a.userID(r); err != nil {
		return err
	}
	list, err := a.Store.PlayersList(200)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "players": list})
	return nil
}