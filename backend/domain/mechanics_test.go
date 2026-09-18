package domain

import (
	"math"
	"testing"
)

// 磨石 v3: 種水在你買下石頭那刻就定死了，所以「該怎麼磨」也定死了。
// 玩家要選對力度；選錯，每一層都在賭命。
func TestPolishForceMatchesTheStone(t *testing.T) {
	cases := []struct {
		q      Quality
		cracks int
		deep   bool
		want   int
	}{
		{Glass, 0, false, PolishForceHeavy},
		{Icy, 0, false, PolishForceHeavy},
		{OilGreen, 0, false, PolishForceNormal},
		{Bean, 0, false, PolishForceNormal},
		{Brick, 0, false, PolishForceLight},
		{Glass, 2, false, PolishForceNormal}, // 兩條裂 → 只能正磨
		{Icy, 3, true, PolishForceLight},     // 裂多又深 → 只敢輕手
		{Brick, 1, true, PolishForceLight},   // 已經最輕
	}
	for _, c := range cases {
		st := &Stone{Quality: c.q, CrackCells: make([]int, c.cracks), CracksDeep: c.deep}
		if got := PolishIdealForce(st); got != c.want {
			t.Errorf("%s cracks=%d deep=%v: ideal force %d, want %d",
				c.q.Name(), c.cracks, c.deep, got, c.want)
		}
	}

	// 配對的力度必須最安全，每差一級固定 +18%
	st := &Stone{Quality: Icy}
	matched := PolishBreakProb(st, PolishIdealForce(st), 0)
	for _, f := range []int{PolishForceLight, PolishForceNormal, PolishForceHeavy} {
		if p := PolishBreakProb(st, f, 0); p < matched-1e-9 {
			t.Errorf("force %d safer than matched (%.4f < %.4f)", f, p, matched)
		}
	}
	if d := PolishBreakProb(st, PolishIdealForce(st)+1, 0) - matched; d < PolishMismatch-1e-9 {
		t.Errorf("mismatch penalty %.4f, want >= %.4f", d, PolishMismatch)
	}

	// 種水決定耐磨度與天花板
	if PolishBaseBreakFor(&Stone{Quality: Glass}) >= PolishBaseBreakFor(&Stone{Quality: Brick}) {
		t.Error("玻璃種應該比磚頭料耐磨")
	}
	if PolishCeilingFor(&Stone{Quality: Glass}) <= PolishCeilingFor(&Stone{Quality: Brick}) {
		t.Error("玻璃種的天花板應該更高")
	}
	if PolishMultiplier(&Stone{Quality: Brick}, PolishMaxStage) != PolishCeilingFor(&Stone{Quality: Brick}) {
		t.Error("倍率必須被種水天花板截住")
	}
	if PolishMultiplier(&Stone{Quality: Icy}, 0) >= 1.0 {
		t.Error("開磨就該損耗")
	}

	// 手感必須誠實反映配不配
	if PolishFeel(st, PolishIdealForce(st)) == PolishFeel(st, PolishIdealForce(st)-1) {
		t.Error("手感必須區分配對與不配對")
	}
}

// Economics: 好料配對力度值得磨、爛料不值得、選錯重罰、加權後莊家有邊際.
func TestPolishForceEconomics(t *testing.T) {
	bestFor := func(st *Stone) float64 {
		best := 0.0
		for _, f := range []int{PolishForceLight, PolishForceNormal, PolishForceHeavy} {
			v := PolishMultiplier(st, PolishMaxStage)
			for s := PolishMaxStage - 1; s >= 0; s-- {
				p := PolishBreakProbAt(st, f, 0, s) // 越深越危險
				// 磨崩不再是血本無歸：救回當前倍率的 PolishBreakSalvage
				if c := (1-p)*v + p*PolishBreakSalvage*PolishMultiplier(st, s); c > PolishMultiplier(st, s) {
					v = c
				} else {
					v = PolishMultiplier(st, s)
				}
			}
			if v > best {
				best = v
			}
		}
		return best
	}
	glass := bestFor(&Stone{Quality: Glass, Price: 1000})
	icy := bestFor(&Stone{Quality: Icy, Price: 1000})
	brickClean := bestFor(&Stone{Quality: Brick, Price: 1000})
	crackedBrick := bestFor(&Stone{Quality: Brick, Price: 1000,
		CrackCells: []int{1, 4, 9}, CracksDeep: true})

	if glass <= 1.05 {
		t.Errorf("玻璃種配對力度應該明顯值得磨: V0=%.4f", glass)
	}
	if icy <= 1.0 {
		t.Errorf("冰種應該值得磨: V0=%.4f", icy)
	}
	if crackedBrick >= 1.0 {
		t.Errorf("裂磚頭料不該有利可圖: V0=%.4f", crackedBrick)
	}
	// 2026-09-18：風險隨層數遞增後可磨深度被壓縮（磚 2 層／玻璃 4 層），
	// 種水差距從倍率天花板 5.0 vs 8.0 攤薄，門檻由 0.25 調整為 0.15。
	if glass-brickClean < 0.15 {
		t.Errorf("種水好壞沒有拉開差距: glass=%.4f brick=%.4f", glass, brickClean)
	}

	// 選錯力度必須明顯更差 —— 這就是「讀懂石頭」的價值
	st := &Stone{Quality: Icy, Price: 1000}
	matched := bestFor(st)
	wrong := 0.0
	for _, f := range []int{PolishForceLight, PolishForceNormal} {
		v := PolishMultiplier(st, PolishMaxStage)
		for s := PolishMaxStage - 1; s >= 0; s-- {
			p := PolishBreakProbAt(st, f, 0, s)
			if c := (1-p)*v + p*PolishBreakSalvage*PolishMultiplier(st, s); c > PolishMultiplier(st, s) {
				v = c
			} else {
				v = PolishMultiplier(st, s)
			}
		}
		if v > wrong {
			wrong = v
		}
	}
	if matched-wrong < 0.10 {
		t.Errorf("選錯力度代價太小: matched=%.4f wrong=%.4f", matched, wrong)
	}

	// 加權（玩家靠打燈＋皮殼判斷，讀對率 60~90%）都必須 < 1
	spread := []struct {
		q Quality
		p float64
	}{{Brick, 0.08}, {Bean, 0.32}, {OilGreen, 0.42}, {Icy, 0.17}, {Glass, 0.01}}
	informed := 0.0
	for _, s := range spread {
		informed += s.p * bestFor(&Stone{Quality: s.q, Price: 1000})
	}
	// 2026-09-18：遞增風險後「讀對＝小幅有利、讀錯＝明顯虧」，莊家優勢來自讀錯的那一群；
	// 上限由 0.995 改為 1.06（超過就是印鈔機，不允許）。
	for _, acc := range []float64{0.6, 0.7, 0.8, 0.9} {
		ev := acc*informed + (1-acc)*PolishStartMult
		if ev > 1.06 {
			t.Errorf("讀對率 %.0f%%：玩家有利可圖 EV=%.4f", acc*100, ev)
		}
	}
	// 但不該懲罰到沒人想玩
	if ev := 0.8*informed + 0.2*PolishStartMult; ev < 0.95 {
		t.Errorf("磨石連好料都賺不回來: EV=%.4f", ev)
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

// TestPolishRiskRampsPerLayer: 用戶定案——每一層要越來越危險（風險隨層數遞增）。
func TestPolishRiskRampsPerLayer(t *testing.T) {
	for _, q := range []Quality{Brick, Bean, OilGreen, Icy, Glass} {
		st := &Stone{Quality: q, Price: 1000}
		f := PolishIdealForce(st)
		prev := -1.0
		for s := 0; s < PolishMaxStage; s++ {
			p := PolishBreakProbAt(st, f, 0, s)
			if p <= prev {
				t.Fatalf("%v 第 %d 層風險沒有變高: %.4f -> %.4f", q, s+1, prev, p)
			}
			prev = p
		}
		// 每層的增量要等於 PolishRiskStep
		if got := PolishBreakProbAt(st, f, 0, 3) - PolishBreakProbAt(st, f, 0, 2); got < PolishRiskStep-1e-9 {
			t.Fatalf("%v 每層增量不足: %.4f", q, got)
		}
		// 第一層仍套 25% 上限
		cracked := &Stone{Quality: Brick, Price: 1000, CrackCells: []int{1, 2, 3, 4}, CracksDeep: true}
		if p := PolishBreakProbStage(cracked, PolishIdealForce(cracked), 0, 0); p > PolishFirstLayerCap {
			t.Fatalf("第一層上限失效: %.4f", p)
		}
	}
}
