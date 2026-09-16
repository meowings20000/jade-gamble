package domain

import "math"

// perCellOf: 每格價值，至少 1 籌碼（便宜料才不會被整數截斷吃光）。
func perCellOf(s *Stone) int {
	v := s.BaseValue() / ScratchCells
	if v < 1 {
		v = 1
	}
	return v
}

// 刮石 (scratch): 12 cells over the stone. Each reveal accumulates value;
// crack cells slash what you have accumulated. Light hints (70% true) mark
// suspect cells. Stop anytime and sell. Full reveal earns a completeness
// bonus. Technical play: use hints to avoid crack cells.

const ScratchCells = 12

// ScratchShareBonus: revealing all 12 cells (刮到底的獎勵).
const ScratchFullBonus = 1.18

// Crack penalty multipliers applied to accumulated value when a crack cell
// is revealed (deep cracks hurt more).
//
// 2026-09-16 軟化: 原本 0.80/0.55 讓刮石變成陷阱——2 條裂紋只剩 0.74、
// 4 條剩 0.51（49% 機率刮到腰斬）。現在乾淨石頭刮到底有 1.18×，
// 有裂的也只是打折而不是腰斬，實測爆倉率 0%。
const (
	CrackPenalty     = 0.95
	DeepCrackPenalty = 0.88
)

type ScratchState struct {
	Revealed    map[int]bool // cell → revealed
	CrackAt     map[int]bool // cell → is a crack cell (server-only truth)
	DeepAt      map[int]bool // cell → deep crack
	Order       []int        // 開格順序：裂紋是「乘法」，順序會影響累積值
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
		PerCell:  perCellOf(s),
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
	st.Order = append(st.Order, cell)
	if st.CrackAt[cell] {
		pen := CrackPenalty
		if st.DeepAt[cell] {
			pen = DeepCrackPenalty
			hitDeep = true
		}
		st.Accumulated = int(math.Round(float64(st.Accumulated) * pen))
		hitCrack = true
	} else {
		st.Accumulated += st.PerCell
	}
	if len(st.Revealed) == ScratchCells {
		st.Accumulated = int(math.Round(float64(st.Accumulated) * ScratchFullBonus))
		st.Done = true
	}
	return hitCrack, hitDeep
}

// Recompute: 由「開格順序」重算累積值（伺服器權威值）。
//
// 注意：不能用 Reveal() 來 replay —— Reveal 看到 revealed 已 true 就早退，
// 拿它回放已開的格會全部不計，累積值永遠只有最後一格的價值
// （2026-09-16 修掉的正式 bug：刮到底只賠付約 1/12 的價值）。
func (st *ScratchState) Recompute() {
	st.Accumulated = 0
	st.Done = false
	for _, c := range st.Order {
		if st.CrackAt[c] {
			pen := CrackPenalty
			if st.DeepAt[c] {
				pen = DeepCrackPenalty
			}
			st.Accumulated = int(math.Round(float64(st.Accumulated) * pen))
		} else {
			st.Accumulated += st.PerCell
		}
	}
	if len(st.Order) == ScratchCells {
		st.Accumulated = int(math.Round(float64(st.Accumulated) * ScratchFullBonus))
		st.Done = true
	}
}

// SellNow banks the current accumulation (early-stop fee 4%).
func (st *ScratchState) SellNow() int {
	if st.Done {
		return st.Accumulated
	}
	return int(math.Round(float64(st.Accumulated) * 0.96))
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
