package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// 管理員控制臺（2026-09-16 新增）
//
// 誰是管理員：
//  1. 環境變數 ADMIN_DISCORD_IDS（逗號分隔的 Discord ID）— 最權威。
//  2. 第一個用 Discord 登入的真人帳號會自動成為管理員（bootstrap）——
//     這樣你不用先知道自己的 ID 才能進控制臺；進去之後畫面上會顯示你的
//     Discord ID，把它填進 .env 的 ADMIN_DISCORD_IDS 就固定住了。
//
// 能做的事：派/收回籌碼、全服紅包、發佈活動公告、看營運數字、清測試帳號。

// isAdmin: 這個玩家是不是管理員。
func (a *API) isAdmin(uid int) bool {
	if uid <= 0 {
		return false
	}
	if a.Store.IsAdmin(uid) {
		return true
	}
	// 環境變數名單：用 Discord ID 比對
	if len(a.AdminDiscordIDs) > 0 {
		if u, err := a.Store.GetUser(uid); err == nil {
			for _, id := range a.AdminDiscordIDs {
				if id != "" && id == u.DiscordID {
					_ = a.Store.AddAdmin(uid, "env")
					return true
				}
			}
		}
	}
	return false
}

// bootstrapAdmin: 第一個真人（Discord 登入）帳號自動當管理員。
func (a *API) bootstrapAdmin(uid int, discordID string) {
	if uid <= 0 || strings.HasPrefix(discordID, "mock:") || discordID == "" {
		return
	}
	if a.Store.AdminCount() > 0 {
		return
	}
	_ = a.Store.AddAdmin(uid, "bootstrap")
}

func (a *API) requireAdmin(w http.ResponseWriter, r *http.Request) (int, error) {
	uid, err := a.userID(r)
	if err != nil {
		return 0, err
	}
	if !a.isAdmin(uid) {
		return 0, errors.New("只有管理員可以進來")
	}
	return uid, nil
}

// adminPanel: GET /api/admin/panel
func (a *API) adminPanel(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	players, err := a.Store.AdminPlayers(400)
	if err != nil {
		return err
	}
	events, err := a.Store.ActiveEvents()
	if err != nil {
		return err
	}
	st, err := a.Store.AdminStats()
	if err != nil {
		return err
	}
	me, _ := a.Store.GetUser(uid)
	myDiscord := ""
	if me != nil {
		myDiscord = me.DiscordID
	}
	writeJSON(w, 200, map[string]any{
		"players": players, "players_listed": len(players), "events": events, "stats": st,
		"my_discord_id": myDiscord,
		"bots":          marketBotNames(),
	})
	return nil
}

// adminGrant: POST /api/admin/grant —— 給/收回某人的籌碼。
func (a *API) adminGrant(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var body struct {
		User   string `json:"user"`
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	if body.Amount == 0 {
		return errors.New("金額不能是 0")
	}
	target, err := a.Store.UserByUsername(strings.TrimSpace(body.User))
	if err != nil {
		return errors.New("找不到這個玩家（名稱要完全一樣）")
	}
	if target.ID == uid && body.Amount < 0 {
		return errors.New("不能從自己身上扣籌碼")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, target.ID, body.Amount)
		if err != nil {
			return err
		}
		bal = b
		return a.Store.LogAdminActionTx(tx, uid, "grant", target.ID, body.Amount, body.Reason)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal,
		"message": "已調整 " + target.Username + " 的籌碼"})
	return nil
}

// adminGiveAll: POST /api/admin/giveall —— 全服紅包。
func (a *API) adminGiveAll(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var body struct {
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	if body.Amount <= 0 {
		return errors.New("金額要大於 0")
	}
	if body.Amount > 1000000 {
		return errors.New("一次最多 1,000,000")
	}
	var n int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		c, err := a.Store.GrantAllTx(tx, body.Amount)
		if err != nil {
			return err
		}
		n = c
		return a.Store.LogAdminActionTx(tx, uid, "giveall", 0, body.Amount, body.Reason)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "players": n,
		"message": "已發紅包給 " + strconv.Itoa(n) + " 位玩家"})
	return nil
}

// adminEvent: POST /api/admin/event —— 發佈活動公告。
func (a *API) adminEvent(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var body struct {
		Title  string `json:"title"`
		Body   string `json:"body"`
		Hours  int    `json:"hours"`
		Amount int    `json:"amount"` // 可選：順便發紅包
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	body.Title = strings.TrimSpace(body.Title)
	body.Body = strings.TrimSpace(body.Body)
	if body.Title == "" {
		return errors.New("活動要有標題")
	}
	if body.Hours <= 0 {
		body.Hours = 24
	}
	if body.Hours > 24*30 {
		return errors.New("最多 30 天")
	}
	var id int
	var paid int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		i, err := a.Store.CreateEventTx(tx, uid, body.Title, body.Body, body.Hours)
		if err != nil {
			return err
		}
		id = i
		if body.Amount > 0 {
			c, err := a.Store.GrantAllTx(tx, body.Amount)
			if err != nil {
				return err
			}
			paid = c
		}
		return a.Store.LogAdminActionTx(tx, uid, "event", 0, body.Amount, body.Title)
	}); err != nil {
		return err
	}
	msg := "活動已發佈：" + body.Title
	if paid > 0 {
		msg += "（同時發紅包給 " + strconv.Itoa(paid) + " 人）"
	}
	writeJSON(w, 200, map[string]any{"ok": true, "event_id": id, "players": paid, "message": msg})
	return nil
}

// adminEventDelete: POST /api/admin/event/delete
func (a *API) adminEventDelete(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var body struct {
		ID int `json:"id"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	if err := a.Store.CloseEvent(body.ID); err != nil {
		return err
	}
	_ = a.Store.WithTx(func(tx *store.Tx) error {
		return a.Store.LogAdminActionTx(tx, uid, "event_end", 0, 0, strconv.Itoa(body.ID))
	})
	writeJSON(w, 200, map[string]any{"ok": true, "message": "活動已下架"})
	return nil
}

// events: GET /api/events —— 所有玩家都看得到的活動橫幅。
func (a *API) events(w http.ResponseWriter, r *http.Request) error {
	if _, err := a.userID(r); err != nil {
		return err
	}
	events, err := a.Store.ActiveEvents()
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"events": events})
	return nil
}

// adminCleanup: POST /api/admin/cleanup —— 清掉測試帳號（mock: 開頭）。
func (a *API) adminCleanup(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.requireAdmin(w, r)
	if err != nil {
		return err
	}
	var n int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		c, err := a.Store.DeleteMockUsersTx(tx)
		if err != nil {
			return err
		}
		n = c
		return a.Store.LogAdminActionTx(tx, uid, "cleanup", 0, 0, "")
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "deleted": n,
		"message": "已清掉 " + strconv.Itoa(n) + " 個測試帳號"})
	return nil
}

// marketBotNames: 控制臺顯示用的 bot 名單（眼力＝出手價相對真值的倍率）。
func marketBotNames() []map[string]any {
	out := []map[string]any{}
	for _, b := range domain.MarketBots {
		out = append(out, map[string]any{
			"name": b.Name, "eye": b.Eye, "sticky_mins": b.StickyMins, "appetite": b.Appetite,
		})
	}
	return out
}
