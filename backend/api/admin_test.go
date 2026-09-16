package api_test

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"jade-gamble/backend/api"
	"jade-gamble/backend/store"
)

// setupAdmin: 用 .env 風格的管理員名單建一個站台（Discord ID "999" 是管理員）。
func setupAdmin(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	a := &api.API{
		Store:           st,
		BaseURL:         "http://test",
		MockAuth:        true,
		AdminDiscordIDs: []string{"999"},
	}
	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)
	return srv, st
}

// 非管理員一律進不來。
func TestAdminRequiresAdmin(t *testing.T) {
	srv, _ := setupAdmin(t)
	peon := newClient(t, srv, "peon")

	for _, ep := range []string{"/api/admin/panel"} {
		if code, _ := peon.doErr("GET", ep, nil); code < 300 {
			t.Fatalf("%s 竟然讓非管理員進來了", ep)
		}
	}
	for _, ep := range []string{"/api/admin/grant", "/api/admin/giveall", "/api/admin/event", "/api/admin/cleanup"} {
		if code, _ := peon.doErr("POST", ep, map[string]any{"user": "peon", "amount": 100, "title": "x"}); code < 300 {
			t.Fatalf("%s 竟然讓非管理員動作了", ep)
		}
	}
	// 非管理員的 /api/me 不該出現 is_admin
	if me := peon.do("GET", "/api/me", nil); me["is_admin"] == true {
		t.Fatal("普通玩家被標成管理員")
	}
}

// 名單上的 Discord ID 自動取得管理員權限，並且各種動作真的生效。
func TestAdminActionsWork(t *testing.T) {
	srv, st := setupAdmin(t)
	boss := newClient(t, srv, "boss")
	sink := newClient(t, srv, "villager")

	// 測試環境是 mock 登入，所以直接把 boss 加進管理員名單模擬 .env 名單命中
	u, err := st.UserByUsername("boss")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddAdmin(u.ID, "test"); err != nil {
		t.Fatal(err)
	}
	if !st.IsAdmin(u.ID) {
		t.Fatal("AddAdmin 沒生效")
	}
	if me := boss.do("GET", "/api/me", nil); me["is_admin"] != true {
		t.Fatalf("/api/me 沒帶 is_admin: %v", me)
	}

	panel := boss.do("GET", "/api/admin/panel", nil)
	if panel["stats"] == nil || panel["players"] == nil {
		t.Fatalf("panel 內容不完整: %v", panel)
	}
	if len(panel["bots"].([]any)) == 0 {
		t.Fatal("panel 沒有 bot 名單")
	}

	// 派籌碼
	before := chips(t, sink)
	boss.do("POST", "/api/admin/grant", map[string]any{"user": "villager", "amount": 7777, "reason": "test"})
	if got := chips(t, sink); got != before+7777 {
		t.Fatalf("派籌碼沒生效: %d -> %d", before, got)
	}
	// 收回籌碼
	boss.do("POST", "/api/admin/grant", map[string]any{"user": "villager", "amount": -7777, "reason": "undo"})
	if got := chips(t, sink); got != before {
		t.Fatalf("收回籌碼沒生效: %d -> %d", before, got)
	}
	// 不能派給不存在的人
	if code, _ := boss.doErr("POST", "/api/admin/grant", map[string]any{"user": "ghost", "amount": 100}); code < 300 {
		t.Fatal("派給不存在的玩家竟然成功")
	}

	// 全服紅包
	b1, v1 := chips(t, boss), chips(t, sink)
	boss.do("POST", "/api/admin/giveall", map[string]any{"amount": 500, "reason": "紅包"})
	if chips(t, boss) != b1+500 || chips(t, sink) != v1+500 {
		t.Fatal("全服紅包沒發到所有人")
	}

	// 發活動 → 所有玩家（含非管理員）都看得到
	ev := boss.do("POST", "/api/admin/event", map[string]any{
		"title": "今晚九點磨石祭", "body": "全服磨石祭，來玩", "hours": 6,
	})
	events := sink.do("GET", "/api/events", nil)["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("活動沒有出現在公告: %v", events)
	}
	if events[0].(map[string]any)["title"] != "今晚九點磨石祭" {
		t.Fatalf("活動標題不對: %v", events[0])
	}
	// 下架
	boss.do("POST", "/api/admin/event/delete", map[string]any{"id": int(ev["event_id"].(float64))})
	if got := sink.do("GET", "/api/events", nil)["events"].([]any); len(got) != 0 {
		t.Fatalf("活動下架後還在: %v", got)
	}
}

// 沒登入的人不能看活動公告（避免匿名探測）。
func TestEventsRequireLogin(t *testing.T) {
	srv, _ := setupAdmin(t)
	anon := &client{t: t, srv: srv}
	if code, _ := anon.doErr("GET", "/api/events", nil); code < 300 {
		t.Fatal("未登入竟然看得到活動")
	}
}

// 清測試帳號：只清 mock: 帳號，真人留著。
func TestAdminCleanupMockUsers(t *testing.T) {
	srv, st := setupAdmin(t)
	boss := newClient(t, srv, "boss")
	newClient(t, srv, "junk_one")
	newClient(t, srv, "junk_two")

	// 造一個「真人」帳號（非 mock:）
	real, err := st.CreateUser("123456789", "real_user", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = real
	u, _ := st.UserByUsername("boss")
	if err := st.AddAdmin(u.ID, "test"); err != nil {
		t.Fatal(err)
	}

	res := boss.do("POST", "/api/admin/cleanup", map[string]any{})
	if int(res["deleted"].(float64)) < 3 {
		t.Fatalf("清掉的帳號數不對: %v", res)
	}
	if _, err := st.UserByUsername("junk_one"); err == nil {
		t.Fatal("測試帳號沒被清掉")
	}
	if _, err := st.UserByUsername("real_user"); err != nil {
		t.Fatal("真人帳號被誤刪了")
	}
}
