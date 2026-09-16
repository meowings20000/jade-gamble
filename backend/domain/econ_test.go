package domain

import (
	"math"
	"testing"
)

// Monte-Carlo the shop economy: EV/price must land within ±2% of the
// target GradeEcon.EVMult per grade. This locks the house edge.
func TestShopEVMonteCarlo(t *testing.T) {
	const N = 200_000
	for _, grade := range []ShopGrade{KiloGrade, FeatureGrade, WindowGrade} {
		r := NewDetRand(uint64(grade) + 1)
		total := 0.0
		for i := 0; i < N; i++ {
			s := GenerateStone(grade, r)
			// Value relative to *this stone's* price: the shop prices stones
			// within a band, so EV ratio is what matters.
			total += float64(s.BaseValue()) / float64(s.Price)
		}
		ev := total / N
		target := grade.Econ().EVMult
		if math.Abs(ev-target) > 0.02 {
			t.Errorf("grade %d: EV/price = %.4f, target %.2f (delta %.4f)",
				grade, ev, target, ev-target)
		} else {
			t.Logf("grade %d (%s): EV/price = %.4f (target %.2f, house edge %.2f%%)",
				grade, grade.Name(), ev, target, (1-ev)*100)
		}
	}
}

// Bianhe egg must be ultra rare, KiloGrade only, and always full green glass.
func TestBianheEgg(t *testing.T) {
	r := NewDetRand(42)
	const N = 2_000_000
	hits := 0
	for i := 0; i < N; i++ {
		s := GenerateStone(KiloGrade, r)
		if s.Egg == "bianhe" {
			hits++
			if s.Quality != Glass || s.Variety != ImperialGreen {
				t.Fatalf("bianhe egg not imperial glass: %s/%s", s.Quality.Name(), s.Variety.Name())
			}
		}
	}
	t.Logf("bianhe rate: %d/%d = %.5f%%", hits, N, float64(hits)/float64(N)*100)
	if hits > N*3/10000 { // sanity: must not exceed ~0.03%
		t.Errorf("bianhe too common: %d hits in %d", hits, N)
	}
}

// B-fake egg: FeatureGrade only, perfect hint, brick inside.
func TestBFakeEgg(t *testing.T) {
	r := NewDetRand(7)
	const N = 500_000
	hits := 0
	for i := 0; i < N; i++ {
		s := GenerateStone(FeatureGrade, r)
		if s.Egg == "b_fake" {
			hits++
			if s.Quality != Brick {
				t.Fatalf("b_fake not brick: %s", s.Quality.Name())
			}
		}
		// Window and Kilo grades must never produce it.
		if g := GenerateStone(WindowGrade, r); g.Egg == "b_fake" {
			t.Fatalf("b_fake appeared in WindowGrade")
		}
		if g := GenerateStone(KiloGrade, r); g.Egg == "b_fake" {
			t.Fatalf("b_fake appeared in KiloGrade")
		}
	}
	t.Logf("b_fake rate: %d/%d = %.4f%%", hits, N, float64(hits)/float64(N)*100)
}

// Variety roll must match the published table within tolerance.
func TestVarietyDistribution(t *testing.T) {
	r := NewDetRand(1234)
	const N = 1_000_000
	counts := map[ColorVariety]int{}
	for i := 0; i < N; i++ {
		v := rollColorVariety(r)
		counts[v]++
	}
	for v, p := range varietyProb {
		got := float64(counts[v]) / N
		if math.Abs(got-p) > 0.0005 {
			t.Errorf("variety %s: got %.5f want %.5f", v.Name(), got, p)
		}
	}
	if c := counts[Base]; c == 0 {
		t.Error("no Base rolled")
	}
}

// Deep crack should cut value; InkGreen hides until opened.
func TestStoneProperties(t *testing.T) {
	r := NewDetRand(99)
	s := &Stone{
		Price: 10_000, Quality: Icy, Variety: Base, CracksDeep: true,
	}
	if v := s.BaseValue(); v != int(10_000*Icy.Multiplier()*0.55) {
		t.Errorf("deep crack value: got %d", v)
	}
	for i := 0; i < 500; i++ {
		g := GenerateStone(WindowGrade, r)
		if g.Variety == InkGreen && !g.InkHidden {
			t.Error("InkGreen without InkHidden")
		}
	}
}
