package domain

// 磨石 (crash-style polish): a rising multiplier ladder. Each advance roll
// risks a break; breaking destroys the stone. Cashing banks BaseValue×mult.
//
// Per-step advance EV is tuned to 0.97 of the current multiplier
// (EV(L[i+1]) = L[i+1]·(1-p[i]) ≈ 0.97·L[i]), so cashing at stage i yields
// EV ≈ 0.97^i — patience is punished, exactly like classic crash games.

type PolishLadder struct {
	Multipliers []float64 // L[0]=1.0 stop-immediately
	BreakProbs  []float64 // p[i] = break chance advancing i → i+1
}

// DefaultPolish: 10 rungs up to ×8, per-step edge 3%.
var DefaultPolish = PolishLadder{
	Multipliers: []float64{1.0, 1.15, 1.35, 1.6, 1.9, 2.3, 2.8, 3.5, 4.5, 6.0, 8.0},
	BreakProbs: []float64{
		0.157, 0.174, 0.182, 0.183, 0.199, 0.204, 0.224, 0.246, 0.273, 0.273,
	},
}

// PolishState is the live run state (persisted per stone in processes).
type PolishState struct {
	Stage int // current rung index
	Alive bool
}

func NewPolishState() *PolishState { return &PolishState{Stage: 0, Alive: true} }

func (p *PolishState) Multiplier() float64 {
	return DefaultPolish.Multipliers[p.Stage]
}

// Advance rolls one more rung. Returns (alive, brokeAtStage).
func (p *PolishState) Advance(r Rand, breakModifier float64) (bool, int) {
	prob := DefaultPolish.BreakProbs[p.Stage] - breakModifier
	if prob < 0 {
		prob = 0
	}
	if r.Float64() < prob {
		p.Alive = false
		return false, p.Stage + 1
	}
	p.Stage++
	return true, -1
}

// CashPayout: stop and bank.
func (p *PolishState) CashPayout(baseValue int) int {
	return int(float64(baseValue) * p.Multiplier())
}
