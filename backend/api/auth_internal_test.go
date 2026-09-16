package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"jade-gamble/backend/store"
)

// 公會白名單：只有指定伺服器的成員可以登入。
func TestDiscordGuildWhitelist(t *testing.T) {
	const guild = "1544329028012474458"
	in := []byte(`[{"id":"999","name":"別的伺服器"},{"id":"1544329028012474458","name":"猪猪岛"}]`)
	ok, err := guildListContains(in, guild)
	if err != nil || !ok {
		t.Fatalf("member of the guild must pass: ok=%v err=%v", ok, err)
	}
	out := []byte(`[{"id":"999","name":"別的伺服器"}]`)
	if ok, err = guildListContains(out, guild); err != nil || ok {
		t.Fatalf("non-member must be rejected: ok=%v err=%v", ok, err)
	}
	if _, err := guildListContains([]byte(`not json`), guild); err == nil {
		t.Fatal("malformed guild list must error, not silently allow")
	}
	// Discord 回 [] = 這個帳號沒加入任何伺服器 → 拒絕
	if ok, err = guildListContains([]byte(`[]`), guild); err != nil || ok {
		t.Fatal("empty guild list must be rejected")
	}
}

// status 必須告訴前端有沒有開公會限制，而授權網址必須帶 guilds scope。
func TestDiscordGuildRestrictionWiring(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	a := &API{
		Store:               st,
		BaseURL:             "http://test",
		MockAuth:            true,
		DiscordClientID:     "1549397605623275520",
		DiscordClientSecret: "secret",
		DiscordRedirectURI:  "https://example.test/api/auth/discord/callback",
		DiscordGuildID:      "1544329028012474458",
	}
	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)

	var status map[string]any
	resp, err := http.Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(resp.Body).Decode(&status)
	_ = resp.Body.Close()
	if status["discord_guild_restricted"] != true {
		t.Fatalf("status must report the guild restriction: %v", status)
	}

	// 不要自動跟隨 redirect，看 authorize 網址本身
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	authResp, err := client.Get(srv.URL + "/api/auth/discord")
	if err != nil {
		t.Fatal(err)
	}
	_ = authResp.Body.Close()
	loc := authResp.Header.Get("Location")
	if !strings.Contains(loc, "scope=identify") || !strings.Contains(loc, "guilds") {
		t.Fatalf("authorize URL must request the guilds scope: %s", loc)
	}
	if !strings.Contains(loc, "client_id=1549397605623275520") {
		t.Fatalf("authorize URL must carry the client id: %s", loc)
	}

	// 沒有設 guild → 不限制
	a.DiscordGuildID = ""
	srv2 := httptest.NewServer(a.Routes())
	t.Cleanup(srv2.Close)
	resp2, err := http.Get(srv2.URL + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	var status2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&status2)
	_ = resp2.Body.Close()
	if status2["discord_guild_restricted"] != false {
		t.Fatalf("status must report no restriction when unset: %v", status2)
	}
}
