package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"jade-gamble/backend/store"
)

// discordCallback exchanges ?code for a Discord user, upserts, sets session.
func (a *API) discordCallback(w http.ResponseWriter, r *http.Request) error {
	if a.DiscordClientID == "" {
		return fmt.Errorf("Discord 登入未設定")
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		return fmt.Errorf("缺少 code")
	}
	// Verify state if the cookie exists.
	if sc, err := r.Cookie("oauth_state"); err == nil {
		if st := r.URL.Query().Get("state"); st == "" || st != sc.Value {
			return fmt.Errorf("state 不符")
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
		return fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("token exchange status %d: %s", resp.StatusCode, string(b))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		return fmt.Errorf("token decode failed")
	}

	ureq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uresp, err := http.DefaultClient.Do(ureq)
	if err != nil {
		return fmt.Errorf("fetch user: %w", err)
	}
	defer uresp.Body.Close()
	if uresp.StatusCode != 200 {
		return fmt.Errorf("fetch user status %d", uresp.StatusCode)
	}
	var user struct {
		ID       string `json:"id"`
		Username string `json:"global_name"`
		Fallback string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	if err := json.NewDecoder(uresp.Body).Decode(&user); err != nil {
		return fmt.Errorf("user decode: %w", err)
	}
	if user.Username == "" {
		user.Username = user.Fallback
	}
	avatar := ""
	if user.Avatar != "" {
		avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
	}
	return a.completeLogin(w, user.ID, user.Username, avatar)
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

// completeLogin upserts the user and sets the session cookie.
func (a *API) completeLogin(w http.ResponseWriter, discordID, username, avatar string) error {
	u, err := a.Store.GetUserByDiscordID(discordID)
	if err != nil {
		u, err = a.Store.CreateUser(discordID, username, avatar)
		if err != nil {
			return err
		}
	}
	tok, err := a.Store.CreateSession(u.ID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600,
	})
	writeJSON(w, 200, map[string]any{"ok": true, "username": username})
	return nil
}

var _ = store.ErrNotFound
