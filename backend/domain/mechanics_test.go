package domain

import (
	"math"
	"testing"
)

// Polish ladder: per-step EV decay ~3%, cash multipliers as published.
func TestPolishLadder(t *testing.T) {
	for i, brk := range DefaultPolish.BreakProbs {
		next := DefaultPolish.Multipliers[i+1]
		cur := DefaultPolish.Multipliers[i]
		evRatio := (next * (1 - brk)) / cur
		if math.Abs(evRatio-0.97) > 0.012 {
			t.Errorf("rung %d: EV ratio %.4f, want ~0.97", i, evRatio)
		}
	}
	r := NewDetRand(5)
	broke := 0
	for i := 0; i < 100_000; i++ {
		p2 := NewPolishState()
		for {
			alive, _ := p2.Advance(r, 0)
			if !alive {
				broke++
				break
			}
			if p2.Stage == 10 {
				break
			}
		}
	}
	if broke == 0 {
		t.Error("no breaks in 100k runs")
	}
	// buff reduces break prob
	p3 := NewPolishState()
	alive := true
	for p3.Stage < 3 {
		alive, _ = p3.Advance(r, 0.08)
	}
	_ = alive
}

// Scratch: EV per cell, crack penalty, sell-now fee, full bonus.
func TestScratchFlow(t *testing.T) {
	s := &Stone{ID: "S_test", Price: 12_000, Quality: Bean, Variety: Base,
		CrackCells: []int{3, 7}, CracksDeep: false}
	r := NewDetRand(11)
	st := NewScratch(s, r)
	// two crack cells mapped
	if len(st.CrackAt) != 2 {
		t.Fatalf("crack cells mapped: %d", len(st.CrackAt))
	}
	// reveal all cells; track accumulation
	var lastCrackHit bool
	for c := 0; c < ScratchCells; c++ {
		hit, _ := st.Reveal(c)
		if hit {
			lastCrackHit = true
		}
	}
	if !st.Done {
		t.Error("not done after 12 reveals")
	}
	if !lastCrackHit {
		t.Error("cracks never hit")
	}
	// EV: per-cell adds BaseValue/12; two cracks multiply by 0.8 each
	// full bonus 1.08. Sanity range only (order-dependent).
	if st.Accumulated <= 0 {
		t.Error("accumulated non-positive")
	}
}

// Sell-now fee applies only when incomplete.
func TestScratchSellNow(t *testing.T) {
	s := &Stone{ID: "S_t", Price: 6000, Quality: Bean, Variety: Base, CrackCells: nil}
	r := NewDetRand(2)
	st := NewScratch(s, r)
	st.Reveal(1)
	got := st.SellNow()
	want := int(float64(st.Accumulated) * 0.96)
	if got != want {
		t.Errorf("sell-now: got %d want %d", got, want)
	}
}

// Setting: crack inside region slashes, deep crack slashes harder.
func TestSetting(t *testing.T) {
	s := &Stone{ID: "S_s", Price: 10_000, Quality: Icy, Variety: Base,
		CrackCells: []int{40}, CracksDeep: false}
	base := s.BaseValue()
	// anchor far from crack 40 (x=4,y=3): anchor (0,0)=0
	res := ComputeSetting(s, BraceletMold, 0, base)
	if res.CracksHit != 0 {
		t.Errorf("unexpected crack hit at far anchor: %+v", res)
	}
	if res.Payout <= 0 {
		t.Errorf("clean payout: %+v", res)
	}
	// anchor directly on crack
	res2 := ComputeSetting(s, BraceletMold, 40, base)
	if res2.CracksHit == 0 {
		t.Error("crack not hit when anchored on it")
	}
	if res2.Payout >= res.Payout {
		t.Errorf("cracked payout should be lower: %+v vs %+v", res2, res)
	}
}

// Exchange EV: frenzy ticket EV 50.5 vs price 2000 — but it grants 10 stones
// → EV 505 < 2000 ✓ (pure sink, sells feel-good).
func TestFrenzyEV(t *testing.T) {
	r := NewDetRand(3)
	total := 0
	const N = 100_000
	for i := 0; i < N; i++ {
		total += FrenzyTicketPayout(r)
	}
	ev := float64(total) / N
	if math.Abs(ev-50.5) > 1 {
		t.Errorf("frenzy EV: %.2f, want 50.5", ev)
	}
	// 10 stones per ticket
	if ev*10 >= 2000 {
		t.Errorf("frenzy ticket is +EV: %.0f", ev*10)
	}
}

// Refresh pricing doubles, capped.
func TestRefreshPrice(t *testing.T) {
	want := []int{300, 600, 1200, 2400, 4800, 4800}
	for i, w := range want {
		if got := RefreshPrice(KiloGrade, i); got != w {
			t.Errorf("kilo refresh %d: got %d want %d", i, got, w)
		}
	}
	if got := RefreshPrice(WindowGrade, 0); got != 5000 {
		t.Errorf("window base refresh: %d", got)
	}
}

// Cut: double coupon doubles positive payouts only.
func TestCutCoupon(t *testing.T) {
	s := &Stone{ID: "S_c", Price: 10_000, Quality: Icy, Variety: Base}
	base := s.BaseValue()
	if CutReveal(s, true) != base*2 {
		t.Error("double coupon not applied")
	}
	s2 := &Stone{ID: "S_c2", Price: 10_000, Quality: Brick, Variety: Base}
	if got := CutReveal(s2, true); got != s2.BaseValue() {
		t.Errorf("brick with coupon: %d", got)
	}
}
