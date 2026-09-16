package domain

// ClassicBet: traditional mode (传统模式) economics.
//
// The player stakes chips upfront; the server rolls ONE blind stone whose
// price == the stake, then the player may only cut / polish / scratch it.
// No resale, no setting, no hints.
//
// The grade is picked by the stake band (higher stakes → richer bands):
//   100..1,999   → kilo distribution
//   2,000..19,999→ feature distribution
//   20,000+      → window distribution
// The interior is rolled from the same table as the matching shelf grade,
// so the house edge is identical to shop pricing (3.2% / 6.7% / 9.4%).

import "errors"

var (
	ErrInvalidStake = errors.New("赌资需在 100 ~ 1,000,000 之间")
)

const (
	MinStake = 100
	MaxStake = 1_000_000
)

// StakeBand returns the shop-grade distribution used for a classic stake.
func (s *Stone) StakeBand() ShopGrade { return StakeBand(s.Price) }

// StakeBand returns the shop-grade distribution used for a classic stake.
func StakeBand(stake int) ShopGrade {
	switch {
	case stake < 2_000:
		return KiloGrade
	case stake < 20_000:
		return FeatureGrade
	default:
		return WindowGrade
	}
}

// GenerateClassicStone rolls a traditional-mode stone for a stake.
// The stone's Price is set to the stake so BaseValue EV == shop EV.
func GenerateClassicStone(stake int, r Rand) *Stone {
	if stake < MinStake || stake > MaxStake {
		return nil // caller validates first
	}
	g := StakeBand(stake)
	s := GenerateStone(g, r)
	s.Price = stake
	s.Origin = "classic"
	s.LightHint = "" // traditional mode: no flashlight report
	return s
}
