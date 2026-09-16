package domain

// 磨石 (crash-style polish) — v2: the ladder now bends with the STONE.
//
// Before v2 the break chance was a fixed table, so polishing was a pure
// crash game: a 磚頭料 and a 玻璃種 walked the same ladder, and the rung
// labels ("冰種初成", "玻璃種極品") were pure decoration. Only the payout
// touched the stone (via BaseValue).
//
// Now the risk profile reads the stone:
//   - every crack cell adds break risk (a cracked stone splits on the wheel)
//   - a deep crack adds a lot
//   - 種水 (quality) absorbs risk: 玻璃種 endures, 磚頭料 crumbles
// and every advance reports a 手感 (feel) line that truthfully describes
// the quality band now visible on the polished face — so polishing is how
// you *learn* a stone, the way 切 reveals it in one blow and 刮 reveals it
// cell by cell.
//
// Economics (pinned by TestPolishLadderV2):
//   - MULT[0] = 0.93: starting the wheel costs 7% (a polished face is worth
//     less than a sealed one) — this removes the free option that made
//     polishing weakly dominate cutting.
//   - A player who KNOWS the stone and plays optimally earns ~0.979× the
//     cut value on a KiloGrade quality spread (house edge ≈2.1%); a player
//     who guesses badly (brick with cracks) sits at 0.93 or loses the lot.

type PolishLadder struct {
	Multipliers []float64 // M[0]=0.93 (start), up to M[10]=5.5
	BreakProbs  []float64 // base break chance advancing i → i+1
}

// DefaultPolish: 11 rungs, ×0.93 start → ×5.5 top, ~2.1% house edge.
//
//nolint:gochecknoglobals
var DefaultPolish = PolishLadder{
	Multipliers: []float64{0.93, 1.111, 1.327, 1.585, 1.893, 2.262, 2.702, 3.227, 3.855, 4.604, 5.5},
	BreakProbs: []float64{
		0.1369, 0.1518, 0.1587, 0.1596, 0.1736, 0.1779, 0.1954, 0.2145, 0.2381, 0.2381,
	},
}

// Risk tuning: quality relief and crack exposure.
const (
	PolishCrackRisk      = 0.03 // per crack cell
	PolishDeepCrackRisk  = 0.10 // extra for a deep crack
	PolishRiskFloor      = 0.02
	PolishRiskCeil       = 0.95
	PolishStartPenaltyID = "polish_start" // MULT[0] documents the 7% entry cost
)

// PolishRiskDelta: how this stone bends the ladder (+ = more likely to break).
// 種水 absorbs risk; 磚頭料 is brittle.
func PolishRiskDelta(st *Stone) float64 {
	if st == nil {
		return 0
	}
	d := float64(len(st.CrackCells)) * PolishCrackRisk
	if st.CracksDeep {
		d += PolishDeepCrackRisk
	}
	switch st.Quality {
	case Brick:
		d += 0.030
	case Bean:
		d += 0.015
	case OilGreen:
		// neutral
	case Icy:
		d -= 0.015
	case Glass:
		d -= 0.030
	}
	return d
}

// PolishState is the live run state (persisted per stone).
type PolishState struct {
	Stage int // current rung index
	Alive bool
}

func NewPolishState() *PolishState { return &PolishState{Stage: 0, Alive: true} }

func (p *PolishState) Multiplier() float64 {
	return DefaultPolish.Multipliers[p.Stage]
}

// BreakProbAt: the real break chance of advancing one rung, for this stone.
func (p *PolishState) BreakProbAt(st *Stone, breakModifier float64) float64 {
	prob := DefaultPolish.BreakProbs[p.Stage] + PolishRiskDelta(st) - breakModifier
	if prob < PolishRiskFloor {
		prob = PolishRiskFloor
	}
	if prob > PolishRiskCeil {
		prob = PolishRiskCeil
	}
	return prob
}

// AdvanceStone rolls one more rung using the stone's own risk profile.
func (p *PolishState) AdvanceStone(st *Stone, r Rand, breakModifier float64) (bool, int) {
	if r.Float64() < p.BreakProbAt(st, breakModifier) {
		p.Alive = false
		return false, p.Stage + 1
	}
	p.Stage++
	return true, -1
}

// Advance keeps the stone-independent roll for callers that only need the
// ladder shape (tests, simulations).
func (p *PolishState) Advance(r Rand, breakModifier float64) (bool, int) {
	return p.AdvanceStone(nil, r, breakModifier)
}

// CashPayout: stop and bank.
func (p *PolishState) CashPayout(baseValue int) int {
	return int(float64(baseValue) * p.Multiplier())
}

// PolishFeel: what the polished face looks like — an honest read of 種水.
// Polishing one rung strips a little more skin, so the read gets sharper.
func PolishFeel(st *Stone, stage int) string {
	if st == nil {
		return "皮殼粗糙，看不出什麼。"
	}
	sharp := stage >= 4
	switch st.Quality {
	case Glass:
		if sharp {
			return "磨面起熒泛光，螢光隱隱流動——這是玻璃種的手感。"
		}
		return "磨面異常細膩，反光有點不尋常。"
	case Icy:
		if sharp {
			return "膠感黏手，透光見底，起冰味了。"
		}
		return "磨面開始發亮，水頭似乎不短。"
	case OilGreen:
		if sharp {
			return "油性上手，色根漸顯。"
		}
		return "磨面見色，油潤感漸出。"
	case Bean:
		if sharp {
			return "磨面見色，但水頭短、發乾。"
		}
		return "皮殼下透出一點色，看起來普通。"
	default: // Brick
		if sharp {
			return "棉絮雜質明顯，磨面粗糙發白——這料沒什麼水頭。"
		}
		return "磨下去發澀，砂感很重。"
	}
}

// PolishLadderTop is the highest multiplier reachable.
func PolishLadderTop() float64 { return DefaultPolish.Multipliers[len(DefaultPolish.Multipliers)-1] }
