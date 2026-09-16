package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// discordCallback exchanges ?code for a Discord user, upserts, sets session.
// loginFail: 登入失敗一律導回登入頁並帶原因，不要丟使用者一個 400 空白頁。
func (a *API) loginFail(w http.ResponseWriter, r *http.Request, code string) error {
	http.Redirect(w, r, "/?login_error="+code, http.StatusFound)
	return nil
}

func (a *API) discordCallback(w http.ResponseWriter, r *http.Request) error {
	if a.DiscordClientID == "" {
		return a.loginFail(w, r, "disabled")
	}
	// 使用者在 Discord 按了「拒絕」
	if r.URL.Query().Get("error") != "" {
		return a.loginFail(w, r, "denied")
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		return a.loginFail(w, r, "nocode")
	}
	// Verify state if the cookie exists.
	if sc, err := r.Cookie("oauth_state"); err == nil {
		if st := r.URL.Query().Get("state"); st == "" || st != sc.Value {
			return a.loginFail(w, r, "state")
		}
	}

	form := url.Values{}
	form.Set("client_id", a.DiscordClientID)
	form.Set("client_secret", a.DiscordClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", a.DiscordRedirectURI)

	resp, err := http.Post("https://discord.com/api/oauth2/token",
		"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return a.loginFail(w, r, "token")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 2048))
		return a.loginFail(w, r, "token")
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		return a.loginFail(w, r, "token")
	}

	ureq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uresp, err := http.DefaultClient.Do(ureq)
	if err != nil {
		return a.loginFail(w, r, "user")
	}
	defer uresp.Body.Close()
	if uresp.StatusCode != 200 {
		return a.loginFail(w, r, "user")
	}
	var user struct {
		ID       string `json:"id"`
		Username string `json:"global_name"`
		Fallback string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	if err := json.NewDecoder(uresp.Body).Decode(&user); err != nil {
		return a.loginFail(w, r, "user")
	}
	if user.Username == "" {
		user.Username = user.Fallback
	}
	avatar := ""
	if user.Avatar != "" {
		avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
	}
	// 公會白名單：不在指定伺服器的人，登不進來
	if a.DiscordGuildID != "" {
		in, err := userInGuild(tok.AccessToken, a.DiscordGuildID)
		if err != nil {
			return a.loginFail(w, r, "guildcheck")
		}
		if !in {
			return a.loginFail(w, r, "guild")
		}
	}
	u, err := a.establishSession(w, user.ID, user.Username, avatar)
	if err != nil {
		return a.loginFail(w, r, "session")
	}
	// 第一個用 Discord 登入的真人自動成為管理員（控制臺 bootstrap）
	a.bootstrapAdmin(u.ID, user.ID)
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

// guildListContains: 這個使用者的伺服器清單裡有沒有 guildID。
// 抽成純函式，測試不需要連 Discord。
func guildListContains(raw []byte, guildID string) (bool, error) {
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return false, err
	}
	for _, g := range list {
		if g.ID == guildID {
			return true, nil
		}
	}
	return false, nil
}

// userInGuild asks Discord which guilds this token's user belongs to.
func userInGuild(accessToken, guildID string) (bool, error) {
	req, _ := http.NewRequest("GET", "https://discord.com/api/v10/users/@me/guilds", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("guild list status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, err
	}
	return guildListContains(body, guildID)
}

// mockLogin is a test-only login (creates or reuses a user).
func (a *API) mockLogin(w http.ResponseWriter, r *http.Request) error {
	if !a.MockAuth {
		return fmt.Errorf("mock auth disabled")
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Username == "" {
		body.Username = "mock_user"
	}
	return a.completeLogin(w, "mock:"+body.Username, body.Username, "")
}

// establishSession: 建/取使用者 + 開 session + 種 cookie。**不寫回應本體**，
// 因為 Discord callback 之後要轉址；先前這裡直接寫 JSON，導致轉址前已經送出
// 200，玩家登入後看到一坨 JSON 而不是被帶回遊戲。
func (a *API) establishSession(w http.ResponseWriter, discordID, username, avatar string) (*domain.User, error) {
	u, err := a.Store.GetUserByDiscordID(discordID)
	if err != nil {
		u, err = a.Store.CreateUser(discordID, username, avatar)
		if err != nil {
			return nil, err
		}
	}
	// 名稱/頭像跟著 Discord 更新
	if u.Username != username || u.Avatar != avatar {
		if err := a.Store.UpdateProfile(u.ID, username, avatar); err == nil {
			u.Username, u.Avatar = username, avatar
		}
	}
	tok, err := a.Store.CreateSession(u.ID)
	if err != nil {
		return nil, err
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600,
	})
	return u, nil
}

// completeLogin: 保留給 mock 登入用（要直接回 JSON）。
func (a *API) completeLogin(w http.ResponseWriter, discordID, username, avatar string) error {
	u, err := a.establishSession(w, discordID, username, avatar)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "username": u.Username, "chips": u.Chips})
	return nil
}

var _ = store.ErrNotFound
