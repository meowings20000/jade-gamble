package api_test

import (
	"testing"
)

// Y佬模式: cut resolves immediately with EV-locked rolls; polish is a
// session ladder with cash-out; one live polish session at a time.
func TestYBossMode(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "yboss")

	// invalid stake / choice
	if _, err := post(c, "/api/yboss/bet", map[string]any{"stake": 50, "choice": "cut"}); err == nil {
		t.Fatal("tiny stake accepted")
	}
	if _, err := post(c, "/api/yboss/bet", map[string]any{"stake": 500, "choice": "nope"}); err == nil {
		t.Fatal("invalid choice accepted")
	}

	// cut resolves immediately
	res := c.do("POST", "/api/yboss/bet", map[string]any{"stake": 1000, "choice": "cut"})
	if res["label"] == nil || res["payout"] == nil {
		t.Fatalf("cut response: %v", res)
	}
	mult := res["mult"].(float64)
	payout := int(res["payout"].(float64))
	if payout != int(1000*mult) {
		t.Fatalf("payout %d != 1000×%v", payout, mult)
	}

	// polish session
	res2 := c.do("POST", "/api/yboss/bet", map[string]any{"stake": 1000, "choice": "polish"})
	if res2["rung"].(float64) != 0 || len(res2["ladder"].([]any)) != 7 {
		t.Fatalf("polish start: %v", res2)
	}

	// second polish session blocked while one is live
	if _, err := post(c, "/api/yboss/bet", map[string]any{"stake": 500, "choice": "polish"}); err == nil {
		t.Fatal("second polish session allowed")
	}
	// cut is still allowed during a polish session (independent game)
	res3 := c.do("POST", "/api/yboss/bet", map[string]any{"stake": 100, "choice": "cut"})
	if res3["payout"] == nil {
		t.Fatal("cut blocked during polish session")
	}

	// cash out at rung 0 → stake back
	cash := c.do("POST", "/api/yboss/polish", map[string]any{"advance": false})
	if int(cash["payout"].(float64)) != 1000 {
		t.Fatalf("cash at rung0: %v", cash)
	}

	// after finishing, new session allowed; advance a few times
	c.do("POST", "/api/yboss/bet", map[string]any{"stake": 100, "choice": "polish"})
	alive, rung := true, 0
	for i := 0; i < 7 && alive; i++ {
		adv := c.do("POST", "/api/yboss/polish", map[string]any{"advance": true})
		alive = adv["alive"] == true
		rung = int(adv["rung"].(float64))
		if rung == 6 {
			break // top
		}
	}
	if alive {
		// session still open → cash out and get a payout
		cash := c.do("POST", "/api/yboss/polish", map[string]any{"advance": false})
		if cash["payout"] == nil {
			t.Fatalf("cash after advances: %v", cash)
		}
		want := int(100 * ladderMults[rung])
		if int(cash["payout"].(float64)) != want {
			t.Fatalf("cash payout: got %v want %d (rung %d)", cash["payout"], want, rung)
		}
	} else {
		// crashed → session closed, cash must fail
		if _, err := post(c, "/api/yboss/polish", map[string]any{"advance": false}); err == nil {
			t.Fatal("cash allowed after crash")
		}
	}
}

var ladderMults = []float64{1.0, 1.3, 1.75, 2.4, 3.4, 5.0, 7.5}
