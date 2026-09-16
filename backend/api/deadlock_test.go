package api_test

import (
	"net/http"
	"testing"
	"time"
)

// REGRESSION: shopRefresh must not deadlock the single-conn SQLite pool.
// Before the fix, the refresh transaction called non-tx FillShelfSlot inside
// the closure → permanent hang → every later DB request blocked forever.
// The test calls refresh several times with a watchdog: if it deadlocks the
// suite fails loudly instead of hanging.
func TestShopRefreshNoDeadlock(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "refresher")

	type res struct {
		grade int
	}
	_ = res{}
	run := func() bool {
		done := make(chan struct{})
		go func() {
			// refresh kilo shelf repeatedly — each one charges more
			for i := 0; i < 4; i++ {
				out := c.do("POST", "/api/shop/refresh", map[string]any{"grade": 0})
				if out == nil {
					t.Error("refresh returned error")
				}
				// verify DB still responsive right after (me endpoint touches DB)
				me := c.do("GET", "/api/me", nil)
				if me == nil {
					t.Error("DB unresponsive after refresh — deadlock regression!")
					return
				}
			}
			close(done)
		}()
		select {
		case <-done:
			return true
		case <-time.After(20 * time.Second):
			t.Fatal("shop/refresh deadlocked the DB pool (regression)")
			return false
		}
	}
	if !run() {
		return
	}
	// final sanity: shop view still works and prices escalated
	shop := c.do("GET", "/api/shop", nil)
	g0 := shop["grades"].([]any)[0].(map[string]any)
	if g0["next_refresh"].(float64) < 3000 {
		t.Fatalf("refresh price did not escalate: %v", g0["next_refresh"])
	}
	_ = http.StatusOK
}
