package domain

import (
	"math"
	"testing"
)

// Polish ladder v2: the rung shape is fixed, but each stone bends it.
func TestPolishLadder(t *testing.T) {
	if len(DefaultPolish.Multipliers) != len(DefaultPolish.BreakProbs)+1 {
		t.Fatalf("ladder shape: %d mults vs %d breaks",
			len(DefaultPolish.Multipliers), len(DefaultPolish.BreakProbs))
	}
	if DefaultPolish.Multipliers[0] >= 1.0 {
		t.Errorf("starting the wheel must cost something: MULT[0]=%.3f", DefaultPolish.Multipliers[0])
	}
	// every rung must be breakable (no free lunch on the way up)
	for i, brk := range DefaultPolish.BreakProbs {
		if brk <= 0 {
			t.Errorf("rung %d has zero break chance", i)
		}
	}
	r := NewDetRand(5)
	broke := 0
	for i := 0; i < 50_000; i++ {
		p2 := NewPolishState()
		for {
			alive, _ := p2.AdvanceStone(nil, r, 0)
			if !alive {
				broke++
				break
			}
			if p2.Stage == len(DefaultPolish.Multipliers)-1 {
				break
			}
		}
	}
	if broke == 0 {
		t.Error("no breaks in 50k runs")
	}
	// buff reduces break prob
	noBuff := NewPolishState().BreakProbAt(nil, 0)
	withBuff := NewPolishState().BreakProbAt(nil, 0.08)
	if withBuff >= noBuff {
		t.Errorf("磨石手感 buff must lower the break chance: %.4f vs %.4f", withBuff, noBuff)
	}
}

// Stones must bend the ladder: 種水 endures, cracks (especially deep) split.
func TestPolishReadsTheStone(t *testing.T) {
	clean := &Stone{ID: "S_a", Price: 1000, Quality: Glass, Variety: Base}
	cracked := &Stone{ID: "S_b", Price: 1000, Quality: Brick, Variety: Base,
		CrackCells: []int{1, 4, 9}, CracksDeep: true}
	pg := NewPolishState()
	pb := NewPolishState()
	if pg.BreakProbAt(clean, 0) >= pg.BreakProbAt(cracked, 0) {
		t.Errorf("flawed stone must be riskier: glass/clean %.4f vs brick/cracked %.4f",
			pg.BreakProbAt(clean, 0), pb.BreakProbAt(cracked, 0))
	}
	if d := PolishRiskDelta(cracked) - PolishRiskDelta(clean); d < 0.15 {
		t.Errorf("risk gap too small to matter: %.4f", d)
	}
	// quality relief must be monotone
	prev := 1.0
	for _, q := range []Quality{Brick, Bean, OilGreen, Icy, Glass} {
		d := PolishRiskDelta(&Stone{Quality: q})
		if d > prev+1e-9 {
			t.Errorf("%s risk rose instead of falling: %.4f", q.Name(), d)
		}
		prev = d
	}
	// the feel line must distinguish 種水 even before sharpening
	if PolishFeel(clean, 2) == PolishFeel(cracked, 2) {
		t.Error("feel lines must differ by quality")
	}
}

// The whole point: a player who knows the stone cannot beat cutting by much,
// and an ignorant player loses. Weighted over the KiloGrade quality spread
// the ladder must stay under 1.0 (house edge), while 冰種/玻璃種 are worth
// polishing (>1.0) — that gap IS the appraisal skill.
func TestPolishLadderV2Economics(t *testing.T) {
	best := func(st *Stone) float64 {
		p := &PolishState{Alive: true}
		rungs := len(DefaultPolish.Multipliers)
		// backward induction: V[i] = max(M[i], (1-p_i)·V[i+1])
		v := DefaultPolish.Multipliers[rungs-1]
		for i := rungs - 2; i >= 0; i-- {
			p.Stage = i
			pi := p.BreakProbAt(st, 0)
			cont := (1 - pi) * v
			if cont > DefaultPolish.Multipliers[i] {
				v = cont
			} else {
				v = DefaultPolish.Multipliers[i]
			}
		}
		return v
	}
	spread := []struct {
		q Quality
		p float64 // KiloGrade distribution
	}{
		{Brick, 0.08}, {Bean, 0.32}, {OilGreen, 0.42}, {Icy, 0.17}, {Glass, 0.01},
	}
	weighted := 0.0
	for _, s := range spread {
		weighted += s.p * best(&Stone{Quality: s.q, Price: 1000})
	}
	if weighted > 0.995 {
		t.Errorf("informed player beats cutting: weighted V0=%.4f", weighted)
	}
	if weighted < 0.94 {
		t.Errorf("polish is a trap even for good stones: weighted V0=%.4f", weighted)
	}
	// good stones must be worth polishing, bad ones must not
	if v := best(&Stone{Quality: Icy, Price: 1000}); v <= 1.0 {
		t.Errorf("冰種 should be worth polishing, V0=%.4f", v)
	}
	if v := best(&Stone{Quality: Brick, Price: 1000, CrackCells: []int{2, 5}, CracksDeep: true}); v >= 1.0 {
		t.Errorf("裂磚頭料 must not be profitable, V0=%.4f", v)
	}
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
