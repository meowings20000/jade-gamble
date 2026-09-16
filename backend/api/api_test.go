package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"jade-gamble/backend/api"
	"jade-gamble/backend/store"
)

// setup spins up the full API with mock auth on a temp DB.
func setup(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	a := &api.API{
		Store:    st,
		BaseURL:  "http://test",
		MockAuth: true,
	}
	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)
	return srv, st
}

type client struct {
	t      *testing.T
	srv    *httptest.Server
	cookie string
}

func (c *client) do(method, path string, body interface{}) map[string]any {
	c.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	// note: bytes.NewReader(nil) means empty body; fine for GET
	req, err := http.NewRequest(method, c.srv.URL+path, rdr)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Cookie", c.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Cookies() != nil {
		for _, ck := range resp.Cookies() {
			if ck.Name == "session" {
				c.cookie = "session=" + ck.Value
			}
		}
	}
	var out map[string]any
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		c.t.Fatalf("%s %s: bad JSON (status %d): %v", method, path, resp.StatusCode, err)
	}
	if resp.StatusCode >= 300 {
		c.t.Fatalf("%s %s: status %d body %v", method, path, resp.StatusCode, out)
	}
	return out
}

func newClient(t *testing.T, srv *httptest.Server, username string) *client {
	c := &client{t: t, srv: srv}
	c.do("POST", "/api/auth/mock", map[string]string{"username": username})
	return c
}

func TestFullFlow(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "alice")

	// status
	st := c.do("GET", "/api/status", nil)
	if st["mock_auth"] != true {
		t.Fatalf("status: %v", st)
	}

	// me: signup bonus
	me := c.do("GET", "/api/me", nil)
	if me["chips"].(float64) != 10000 {
		t.Fatalf("signup chips: %v", me["chips"])
	}

	// shop view: three grades, auto-filled shelves
	shop := c.do("GET", "/api/shop", nil)
	grades := shop["grades"].([]any)
	if len(grades) != 3 {
		t.Fatalf("grades: %d", len(grades))
	}
	g0 := grades[0].(map[string]any)
	items := g0["items"].([]any)
	if len(items) != 8 {
		t.Fatalf("kilo shelf size: %d", len(items))
	}
	// shelf stones must NOT leak quality/variety
	first := items[0].(map[string]any)
	for _, leak := range []string{"quality", "variety", "cracks", "crack_cells"} {
		if _, ok := first[leak]; ok {
			t.Fatalf("shelf card leaks %q", leak)
		}
	}

	// buy the first kilo stone
	stoneID := first["id"].(string)
	price := int(first["price"].(float64))
	buy := c.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})
	if buy["chips"].(float64) != float64(10000-price) {
		t.Fatalf("buy chips: %v", buy["chips"])
	}
	// double-buy must fail (stone removed from shelf)
	_, err := post(c, "/api/shop/buy", map[string]string{"stone_id": stoneID})
	if err == nil {
		t.Fatal("double buy succeeded")
	}

	// inventory has the stone
	inv := c.do("GET", "/api/inventory", nil)
	if len(inv["stones"].([]any)) != 1 {
		t.Fatalf("inventory: %v", inv)
	}

	// cut it
	cut := c.do("POST", "/api/cut", map[string]any{"stone_id": stoneID})
	if _, ok := cut["quality"]; !ok {
		t.Fatalf("cut response: %v", cut)
	}
	afterCut := int(cut["chips"].(float64))
	me2 := c.do("GET", "/api/me", nil)
	if int(me2["chips"].(float64)) != afterCut {
		t.Fatalf("cut balance mismatch: %d vs %v", afterCut, me2["chips"])
	}
	// stone is used now
	inv2 := c.do("GET", "/api/inventory", nil)
	if len(inv2["stones"].([]any)) != 0 {
		t.Fatalf("inventory after cut: %v", inv2)
	}
}

// post that expects an error returns err non-nil when status >= 300.
func post(c *client, path string, body interface{}) (map[string]any, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Cookie", c.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("status %d: %v", resp.StatusCode, out)
	}
	return out, nil
}

func TestScratchFlow(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "bob")

	shop := c.do("GET", "/api/shop", nil)
	grades := shop["grades"].([]any)
	g0 := grades[0].(map[string]any)
	items := g0["items"].([]any)
	stoneID := items[0].(map[string]any)["id"].(string)
	c.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})

	start := c.do("POST", "/api/scratch/start", map[string]string{"stone_id": stoneID})
	if start["done"] != false {
		t.Fatalf("scratch start: %v", start)
	}
	acc := 0.0
	for i := 0; i < 12; i++ {
		resp := c.do("POST", "/api/scratch/reveal", map[string]any{"stone_id": stoneID, "cell": i})
		acc = resp["accumulated"].(float64)
		if resp["done"] == true {
			if i < 11 && resp["kind"] == nil {
				// can complete early if all crack-free cells... actually done
				// requires all 12; but reveal order guarantees 12 calls.
			}
			if i == 11 {
				if resp["payout"] == nil {
					t.Fatalf("done reveal missing payout: %v", resp)
				}
			}
		}
	}
	if acc <= 0 {
		t.Fatalf("accumulated: %v", acc)
	}
}

func TestPolishFlow(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "carol")

	shop := c.do("GET", "/api/shop", nil)
	items := shop["grades"].([]any)[0].(map[string]any)["items"].([]any)
	stoneID := items[0].(map[string]any)["id"].(string)
	c.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})

	start := c.do("POST", "/api/polish/start", map[string]string{"stone_id": stoneID})
	if start["alive"] != true {
		t.Fatalf("polish start: %v", start)
	}
	cashed := false
	for i := 0; i < 10; i++ {
		resp := c.do("POST", "/api/polish/advance", map[string]string{"stone_id": stoneID})
		if resp["alive"] == false {
			// broke: stone gone
			_, err := post(c, "/api/polish/advance", map[string]string{"stone_id": stoneID})
			if err == nil {
				t.Fatal("advance after break succeeded")
			}
			cashed = true // mark done
			break
		}
	}
	if !cashed {
		resp := c.do("POST", "/api/polish/cash", map[string]string{"stone_id": stoneID})
		if resp["payout"] == nil {
			t.Fatalf("cash: %v", resp)
		}
	}
}

func TestMarketFlow(t *testing.T) {
	srv, _ := setup(t)
	alice := newClient(t, srv, "alice2")
	bob := newClient(t, srv, "bob2")

	// alice buys an affordable stone (kilo grade, always <= 2000)
	shop := alice.do("GET", "/api/shop", nil)
	items := shop["grades"].([]any)[0].(map[string]any)["items"].([]any)
	stoneID := items[0].(map[string]any)["id"].(string)
	alice.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})

	// lists it at 1000
	alice.do("POST", "/api/market/list", map[string]any{"stone_id": stoneID, "ask_price": 1000})

	// bob sees it (plus his own private 礦區直送 pool)
	mkt := bob.do("GET", "/api/market", nil)
	listings := mkt["listings"].([]any)
	var l map[string]any
	for _, it := range listings {
		cand := it.(map[string]any)
		if cand["stone_id"].(string) == stoneID {
			l = cand
			break
		}
	}
	if l == nil {
		t.Fatalf("alice's listing not visible to bob: %v", listings)
	}
	if _, ok := l["seller_id"]; ok {
		t.Fatal("market listing leaks seller_id")
	}
	if _, ok := l["seller"]; ok {
		t.Fatal("拍賣必須匿名 — listing leaks seller name")
	}
	// no truth leak
	for _, leak := range []string{"quality", "variety", "crack_cells"} {
		if _, ok := l[leak]; ok {
			t.Fatalf("market listing leaks %q", leak)
		}
	}

	// bob buys it
	bob.do("POST", "/api/market/buy", map[string]any{"listing_id": int(l["id"].(float64))})
	inv := bob.do("GET", "/api/inventory", nil)
	if len(inv["stones"].([]any)) != 1 {
		t.Fatalf("bob inventory after buy: %v", inv)
	}
	// double buy fails
	if _, err := post(bob, "/api/market/buy", map[string]any{"listing_id": int(l["id"].(float64))}); err == nil {
		t.Fatal("double market buy succeeded")
	}
}

func TestExchangeAndLeaderboard(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "dave")

	// buy frenzy ticket
	resp := c.do("POST", "/api/exchange/buy", map[string]string{"key": "frenzy_ticket"})
	pays := resp["payouts"].([]any)
	if len(pays) != 10 {
		t.Fatalf("frenzy payouts: %v", pays)
	}

	// leaderboard
	lb := c.do("GET", "/api/leaderboard?kind=wealth", nil)
	entries := lb["entries"].([]any)
	if len(entries) < 1 {
		t.Fatalf("leaderboard empty: %v", lb)
	}

	// collection view
	col := c.do("GET", "/api/collection", nil)
	if col["varieties"] == nil {
		t.Fatalf("collection: %v", col)
	}
}

func TestRelief(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "eve")

	// drain chips by buying the most expensive shelf stones repeatedly
	for i := 0; i < 30; i++ {
		shop := c.do("GET", "/api/shop", nil)
		grades := shop["grades"].([]any)
		// buy window grade (most expensive)
		wg := grades[2].(map[string]any)
		items := wg["items"].([]any)
		if len(items) == 0 {
			continue
		}
		sid := items[0].(map[string]any)["id"].(string)
		if _, err := post(c, "/api/shop/buy", map[string]string{"stone_id": sid}); err != nil {
			break
		}
	}
	me := c.do("GET", "/api/me", nil)
	if int(me["chips"].(float64)) > 0 {
		// relief should refuse
		if _, err := post(c, "/api/relief", map[string]string{"option": "chips"}); err == nil {
			t.Fatal("relief granted while chips > 0")
		} else {
			return
		}
	}
	// broke: relief works
	resp := c.do("POST", "/api/relief", map[string]string{"option": "chips"})
	if int(resp["chips_given"].(float64)) != 1000 {
		t.Fatalf("relief: %v", resp)
	}
}

func TestStrictJSON(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "frank")

	req, _ := http.NewRequest("POST", srv.URL+"/api/auth/mock",
		strings.NewReader(`{"username":"x"}{"username":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Fatal("trailing JSON accepted")
	}
	_ = c
}

// TestMarketNPCPoolsPrivate: 礦區直送 pools are per-player — two players must
// never see the same pool stone (otherwise "who bought what" leaks where the
// good stones are), and a player cannot buy someone else's pool entry.
func TestMarketNPCPoolsPrivate(t *testing.T) {
	srv, _ := setup(t)
	alice := newClient(t, srv, "npc_alice")
	bob := newClient(t, srv, "npc_bob")

	am := alice.do("GET", "/api/market", nil)["listings"].([]any)
	bm := bob.do("GET", "/api/market", nil)["listings"].([]any)
	if len(am) == 0 || len(bm) == 0 {
		t.Fatalf("pools not stocked: alice=%d bob=%d", len(am), len(bm))
	}
	// every entry so far is an NPC stone
	aset := map[string]bool{}
	for _, it := range am {
		m := it.(map[string]any)
		if m["npc"] != true {
			t.Fatalf("expected npc listing, got %v", m)
		}
		aset[m["stone_id"].(string)] = true
	}
	for _, it := range bm {
		m := it.(map[string]any)
		if aset[m["stone_id"].(string)] {
			t.Fatalf("stone %s appears in BOTH players' pools — pools must be private", m["stone_id"])
		}
	}

	// bob cannot buy alice's pool entry (IDs are visible but scoped)
	aliceID := int(am[0].(map[string]any)["id"].(float64))
	if _, err := post(bob, "/api/market/buy", map[string]any{"listing_id": aliceID}); err == nil {
		t.Fatal("bob bought a pool stone that belongs to alice")
	}

	// alice CAN buy an affordable pool stone; the mine sinks the chips
	before := alice.do("GET", "/api/me", nil)["chips"].(float64)
	buyIC := -1
	ask := 0
	for _, it := range am {
		m := it.(map[string]any)
		a := int(m["ask_price"].(float64))
		if a <= int(before) {
			buyIC = int(m["id"].(float64))
			ask = a
			break
		}
	}
	if buyIC < 0 {
		t.Fatalf("no affordable pool stone for %v chips", before)
	}
	res := alice.do("POST", "/api/market/buy", map[string]any{"listing_id": buyIC})
	if int(res["chips"].(float64)) != int(before)-ask {
		t.Fatalf("pool buy charge: before=%v ask=%d after=%v", before, ask, res["chips"])
	}
	inv := alice.do("GET", "/api/inventory", nil)["stones"].([]any)
	if len(inv) != 1 {
		t.Fatalf("bought pool stone not in inventory: %v", inv)
	}
	// the pool refills on the next market view (still 6)
	am2 := alice.do("GET", "/api/market", nil)["listings"].([]any)
	n := 0
	for _, it := range am2 {
		if it.(map[string]any)["npc"] == true {
			n++
		}
	}
	if n != 6 {
		t.Fatalf("pool after buy: want 6 npc stones, got %d", n)
	}
}
