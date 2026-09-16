package domain

import "testing"

// Classic stone: price == stake, correct band per stake, no hint, origin set.
func TestGenerateClassicStone(t *testing.T) {
	r := NewDetRand(2024)
	cases := []struct {
		stake int
		band  ShopGrade
	}{
		{100, KiloGrade},
		{1_999, KiloGrade},
		{2_000, FeatureGrade},
		{19_999, FeatureGrade},
		{20_000, WindowGrade},
		{1_000_000, WindowGrade},
	}
	for _, c := range cases {
		s := GenerateClassicStone(c.stake, r)
		if s == nil {
			t.Fatalf("stake %d rejected", c.stake)
		}
		if s.Price != c.stake {
			t.Errorf("stake %d: price %d", c.stake, s.Price)
		}
		if s.StakeBand() != c.band {
			t.Errorf("stake %d: band %s want %s", c.stake, s.StakeBand().Name(), c.band.Name())
		}
		if s.Origin != "classic" {
			t.Errorf("stake %d: origin %q", c.stake, s.Origin)
		}
		if s.LightHint != "" {
			t.Errorf("stake %d: classic stone must have no hint", c.stake)
		}
	}
	// bounds
	if GenerateClassicStone(99, r) != nil {
		t.Error("stake below minimum accepted")
	}
	if GenerateClassicStone(1_000_001, r) != nil {
		t.Error("stake above maximum accepted")
	}
}

// Classic EV matches the shop EV of the matching band (same roll table).
func TestClassicEVMonteCarlo(t *testing.T) {
	const N = 100_000
	r := NewDetRand(77)
	stakes := []int{1_000, 10_000, 50_000}
	for _, stake := range stakes {
		total := 0.0
		for i := 0; i < N; i++ {
			s := GenerateClassicStone(stake, r)
			total += float64(s.BaseValue()) / float64(s.Price)
		}
		ev := total / N
		target := StakeBand(stake).Econ().EVMult
		if ev > target+0.03 || ev < target-0.03 {
			t.Errorf("stake %d: EV %.3f outside band target %.2f", stake, ev, target)
		} else {
			t.Logf("stake %d (%s): EV/price %.4f, house edge %.2f%%",
				stake, StakeBand(stake).Name(), ev, (1-ev)*100)
		}
	}
}
