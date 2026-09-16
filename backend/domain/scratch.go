package domain

// 刮石 (scratch): 12 cells over the stone. Each reveal accumulates value;
// crack cells slash what you have accumulated. Light hints (70% true) mark
// suspect cells. Stop anytime and sell. Full reveal earns a completeness
// bonus. Technical play: use hints to avoid crack cells.

const ScratchCells = 12

// ScratchShareBonus: revealing all 12 cells.
const ScratchFullBonus = 1.08

// Crack penalty multipliers applied to accumulated value when a crack cell
// is revealed (deep cracks hurt more).
const (
	CrackPenalty     = 0.80
	DeepCrackPenalty = 0.55
)

type ScratchState struct {
	Revealed    map[int]bool // cell → revealed
	CrackAt     map[int]bool // cell → is a crack cell (server-only truth)
	DeepAt      map[int]bool // cell → deep crack
	Accumulated int          // chips accumulated so far
	PerCell     int          // value per normal cell = BaseValue/12
	Done        bool
}

// NewScratch maps the stone's crack cells onto 12 scratch positions. The
// mapping is seeded from the stone ID so it is stable across reloads.
func NewScratch(s *Stone, r Rand) *ScratchState {
	st := &ScratchState{
		Revealed: map[int]bool{},
		CrackAt:  map[int]bool{},
		DeepAt:   map[int]bool{},
		PerCell:  s.BaseValue() / ScratchCells,
	}
	// Distribute the stone's cracks onto distinct cells.
	perm := PermAny(r, ScratchCells)
	n := len(s.CrackCells)
	if n > ScratchCells-1 { // always leave at least one clean cell
		n = ScratchCells - 1
	}
	for i := 0; i < n; i++ {
		st.CrackAt[perm[i]] = true
	}
	if s.CracksDeep && n > 0 {
		st.DeepAt[perm[0]] = true
	}
	return st
}

// Reveal opens one cell. Returns the crack hit (if any) and new accumulated.
func (st *ScratchState) Reveal(cell int) (hitCrack bool, hitDeep bool) {
	if st.Revealed[cell] || st.Done {
		return false, false
	}
	st.Revealed[cell] = true
	if st.CrackAt[cell] {
		pen := CrackPenalty
		if st.DeepAt[cell] {
			pen = DeepCrackPenalty
			hitDeep = true
		}
		st.Accumulated = int(float64(st.Accumulated) * pen)
		hitCrack = true
	} else {
		st.Accumulated += st.PerCell
	}
	if len(st.Revealed) == ScratchCells {
		st.Accumulated = int(float64(st.Accumulated) * ScratchFullBonus)
		st.Done = true
	}
	return hitCrack, hitDeep
}

// SellNow banks the current accumulation (early-stop fee 4%).
func (st *ScratchState) SellNow() int {
	if st.Done {
		return st.Accumulated
	}
	return int(float64(st.Accumulated) * 0.96)
}

// HintFor returns a hint about a cell: 70% true, 30% a lie (per stone
// LieRate). Truth = "crack" iff the cell IS a crack cell.
func (st *ScratchState) HintFor(cell int, r Rand) string {
	truth := st.CrackAt[cell]
	if r.Float64() < 0.30 {
		truth = !truth
	}
	if truth {
		return "这条线走色发闷，风险大。"
	}
	return "看起来干净，可以下刀。"
}
