package domain

// 套型 (setting/molding): after a stone is fully revealed (cut or scratched
// through), choose a mold and an anchor on the 96-cell grid. The mold covers
// a region; cracks inside the region slash the final value.

type Mold int

const (
	BraceletMold Mold = iota // 手镯 ×1.6, any crack inside → ×0.3
	PeaceBuckle              // 平安扣 ×1.3, crack → ×0.5
	Pendant                  // 吊坠 ×1.15, crack → ×0.6
)

var moldInfo = map[Mold]struct {
	Name      string
	BaseMult  float64
	CrackMult float64
	DeepMult  float64
	Radius    int // region radius in cells (anchor-centered, 12×8 grid)
}{
	BraceletMold: {"手镯", 1.6, 0.30, 0.15, 3},
	PeaceBuckle:  {"平安扣", 1.3, 0.50, 0.25, 2},
	Pendant:      {"吊坠", 1.15, 0.60, 0.30, 1},
}

func (m Mold) Name() string { return moldInfo[m].Name }

// SettingFee: processing charge for one setting (a chip sink).
const SettingFee = 0.05 // of BaseValue

// SetResult is the outcome of one setting attempt.
type SetResult struct {
	Mult      float64
	Payout    int
	CracksHit int
	DeepHit   bool
}

// ComputeSetting scores a mold placement on the 12×8 grid.
func ComputeSetting(s *Stone, mold Mold, anchor int, baseValue int) SetResult {
	info := moldInfo[mold]
	mult := info.BaseMult
	ax, ay := anchor%12, anchor/12
	crackSet := map[int]bool{}
	for _, c := range s.CrackCells {
		crackSet[c] = true
	}
	hits := 0
	deep := false
	if s.CracksDeep {
		// deep crack sits at the first crack cell
		if len(s.CrackCells) > 0 {
			deep = crackNear(s.CrackCells[0], ax, ay, info.Radius)
		}
	}
	for c := range crackSet {
		if crackNear(c, ax, ay, info.Radius) {
			hits++
		}
	}
	if hits > 0 {
		mult *= info.CrackMult
	}
	if deep {
		mult *= info.DeepMult
	}
	fee := int(float64(baseValue) * SettingFee)
	return SetResult{
		Mult:      mult,
		Payout:    int(float64(baseValue)*mult) - fee,
		CracksHit: hits,
		DeepHit:   deep,
	}
}

func crackNear(cell, ax, ay, radius int) bool {
	cx, cy := cell%12, cell/12
	dx, dy := cx-ax, cy-ay
	return dx*dx+dy*dy <= radius*radius
}
