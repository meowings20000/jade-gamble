package api_test

import (
	"testing"
)

// Traditional mode: stake → blind stone → only cut/polish/scratch;
// cannot list on market; one pending classic stone at a time.
func TestClassicMode(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "classicist")

	// invalid stakes rejected
	if _, err := post(c, "/api/classic/bet", map[string]any{"stake": 50}); err == nil {
		t.Fatal("stake below minimum accepted")
	}
	if _, err := post(c, "/api/classic/bet", map[string]any{"stake": 2_000_000}); err == nil {
		t.Fatal("stake above maximum accepted")
	}

	// valid bet
	res := c.do("POST", "/api/classic/bet", map[string]any{"stake": 1000})
	if res["chips"].(float64) != 9000 {
		t.Fatalf("after bet: %v", res["chips"])
	}
	stoneID := res["stone_id"].(string)

	// no hint in response
	if _, ok := res["hint"]; ok {
		t.Fatal("classic bet response leaks hint")
	}

	// second bet while pending must fail
	if _, err := post(c, "/api/classic/bet", map[string]any{"stake": 500}); err == nil {
		t.Fatal("second classic bet allowed while one pending")
	}

	// inventory shows it as classic
	inv := c.do("GET", "/api/inventory", nil)
	stone := inv["stones"].([]any)[0].(map[string]any)
	if stone["origin"] != "classic" {
		t.Fatalf("inventory origin: %v", stone["origin"])
	}
	if stone["hint"] != "" {
		t.Fatal("classic stone leaks hint in inventory")
	}

	// cannot list on market
	if _, err := post(c, "/api/market/list", map[string]any{"stone_id": stoneID, "ask_price": 100}); err == nil {
		t.Fatal("classic stone listed on market")
	}

	// cut it (one of the three allowed paths)
	cut := c.do("POST", "/api/cut", map[string]any{"stone_id": stoneID})
	if _, ok := cut["quality"]; !ok {
		t.Fatalf("cut failed: %v", cut)
	}
	if cut["payout"].(float64) < 0 {
		t.Fatal("negative payout")
	}

	// after processing, a new bet is allowed again
	res2 := c.do("POST", "/api/classic/bet", map[string]any{"stake": 100})
	if res2["stone_id"] == nil {
		t.Fatal("bet after processing rejected")
	}
}
