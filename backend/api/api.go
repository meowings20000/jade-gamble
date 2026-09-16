package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jade-gamble/backend/store"
)

// API holds shared dependencies for handlers.
type API struct {
	Store               *store.Store
	BaseURL             string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string
	// DiscordGuildID: 只有這個伺服器的成員可以登入（空 = 不限制）
	DiscordGuildID string
	// MockAuth enables the test-only mock Discord flow (default off in prod).
	MockAuth bool
}

// routes registers everything on a mux.
func (a *API) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/auth/discord", a.handler(a.discordAuthRedirect))
	mux.HandleFunc("GET /api/auth/discord/callback", a.handler(a.discordCallback))
	if a.MockAuth {
		mux.HandleFunc("POST /api/auth/mock", a.handler(a.mockLogin))
	}
	mux.HandleFunc("POST /api/auth/logout", a.handler(a.logout))
	mux.HandleFunc("GET /api/me", a.handler(a.me))
	mux.HandleFunc("GET /api/status", a.handler(a.status))

	mux.HandleFunc("GET /api/shop", a.handler(a.shopView))
	mux.HandleFunc("POST /api/shop/refresh", a.handler(a.shopRefresh))
	mux.HandleFunc("POST /api/shop/buy", a.handler(a.shopBuy))
	mux.HandleFunc("POST /api/shop/light", a.handler(a.shopLightReport))

	mux.HandleFunc("GET /api/inventory", a.handler(a.inventoryList))
	mux.HandleFunc("POST /api/cut", a.handler(a.cut))
	mux.HandleFunc("POST /api/scratch/start", a.handler(a.scratchStart))
	mux.HandleFunc("POST /api/scratch/reveal", a.handler(a.scratchReveal))
	mux.HandleFunc("POST /api/scratch/hint", a.handler(a.scratchHint))
	mux.HandleFunc("POST /api/scratch/sell", a.handler(a.scratchSell))
	mux.HandleFunc("POST /api/polish/start", a.handler(a.polishStart))
	mux.HandleFunc("POST /api/polish/advance", a.handler(a.polishAdvance))
	mux.HandleFunc("POST /api/polish/cash", a.handler(a.polishCash))
	mux.HandleFunc("POST /api/setting", a.handler(a.setting))

	mux.HandleFunc("GET /api/market", a.handler(a.marketList))
	mux.HandleFunc("POST /api/market/list", a.handler(a.marketListStone))
	mux.HandleFunc("POST /api/market/buy", a.handler(a.marketBuy))

	mux.HandleFunc("POST /api/classic/bet", a.handler(a.classicBet))
	mux.HandleFunc("POST /api/yboss/bet", a.handler(a.ybossBet))
	mux.HandleFunc("POST /api/yboss/polish", a.handler(a.ybossPolish))
	mux.HandleFunc("GET /api/exchange", a.handler(a.exchangeView))
	mux.HandleFunc("POST /api/exchange/buy", a.handler(a.exchangeBuy))
	mux.HandleFunc("POST /api/relief", a.handler(a.relief))

	mux.HandleFunc("GET /api/leaderboard", a.handler(a.leaderboard))
	mux.HandleFunc("GET /api/collection", a.handler(a.collection))
	mux.HandleFunc("GET /api/hall", a.handler(a.hallOfFame))
	return mux
}

// status: public site config (login availability etc).
func (a *API) status(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, 200, map[string]any{
		"discord_oauth":            a.DiscordClientID != "",
		"discord_guild_restricted": a.DiscordGuildID != "",
		"mock_auth":                a.MockAuth,
	})
	return nil
}

// discordAuthRedirect sends the user to Discord's OAuth consent page.
func (a *API) discordAuthRedirect(w http.ResponseWriter, r *http.Request) error {
	if a.DiscordClientID == "" {
		return fmt.Errorf("Discord 登入未設定")
	}
	state := newState()
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", MaxAge: 600, HttpOnly: true})
	q := strings.NewReplacer()
	_ = q
	// identify = 誰登入；guilds = 他在哪些伺服器（用來做公會白名單）
	url := fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify%%20guilds&state=%s",
		a.DiscordClientID, url.QueryEscape(a.DiscordRedirectURI), state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie("session"); err == nil {
		_ = a.Store.DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, 200, map[string]string{"ok": "true"})
	return nil
}

// me: current user profile.
func (a *API) me(w http.ResponseWriter, r *http.Request) error {
	uid, ok := a.mustUser(w, r)
	if !ok {
		return nil
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	items, err := a.Store.ListItems(uid)
	if err != nil {
		return err
	}
	disc, err := a.Store.DiscoveredList(uid)
	if err != nil {
		return err
	}
	varieties := []int{}
	for _, v := range disc {
		varieties = append(varieties, int(v))
	}
	writeJSON(w, 200, map[string]any{
		"id": u.ID, "username": u.Username, "avatar": u.Avatar,
		"chips": u.Chips, "collection_score": u.CollectionScore, "title": u.Title,
		"items": items, "discovered": varieties,
	})
	return nil
}

func newState() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
