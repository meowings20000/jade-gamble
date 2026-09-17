package domain

import "math"

// 磨石 v3 — 力度必須配得上種水.
//
// 種水在你買下石頭的那一刻就定死了，所以「這顆料該怎麼磨」其實也已經
// 寫在石頭裡：磚頭料吃不消重磨，玻璃種才扛得住壓力。
//
// 玩家在開磨前選一個力度（輕磨／正磨／重磨），之後不能換。
// 力度與這顆料的匹配度直接決定每一層的爆裂機率：
// 配得上，機器順暢，一路上去；配不上，每一層都在賭命。
// 手感會誠實告訴你配不配——但不會直接告訴你種水是什麼。
//
// EV（TestPolishForceEconomics 鎖定）：
//   - 配對力度：每層 EV/step = 1.20×0.842 = 1.011 → 值得爬，約 ×1.03 期望
//   - 選錯一級：每層 EV/step = 0.664 → 該立刻收手（虧開磨的 7%）
//   - 加權（玩家七成選對，靠打燈與皮殼）≈ 0.99 → 莊家仍有邊際

const (
	// PolishForceLight / Normal / Heavy: 磨石力度.
	PolishForceLight  = 1
	PolishForceNormal = 2
	PolishForceHeavy  = 3

	PolishMismatch  = 0.18 // 每差一級 +18% 爆裂率
	PolishCrackRisk = 0.02 // 每條裂紋
	PolishDeepRisk  = 0.06 // 深裂
	PolishStepGain  = 1.22 // 每層倍率成長（2026-09-17 校準）
	PolishStartMult = 0.93 // 開磨損耗 7%
	// PolishBreakSalvage: 磨崩時救回的比例（2026-09-17 用戶反映「怎麽樣也虧」）。
	// 舊版崩了＝整顆石頭報廢（-100%），一次失手就吃掉上百層的收益；現在救回三成五。
	PolishBreakSalvage = 0.30
	PolishMaxStage     = 10 // 可爬層數
)

// polishBaseBreak: 力度完全配對時，每層的基礎爆裂率——種水就是耐磨度。
// 玻璃種最耐，磚頭料最脆。
var polishBaseBreak = map[Quality]float64{
	Brick:    0.275,
	Bean:     0.254,
	OilGreen: 0.235,
	Icy:      0.218,
	Glass:    0.203,
}

// polishCeiling: 種水決定這顆料能被磨到多高。
var polishCeiling = map[Quality]float64{
	Brick:    5.0,
	Bean:     5.5,
	OilGreen: 6.0,
	Icy:      7.0,
	Glass:    8.0,
}

// PolishBaseBreakFor: 這顆料的基礎爆裂率。
func PolishBaseBreakFor(st *Stone) float64 {
	if st == nil {
		return 0.162
	}
	if v, ok := polishBaseBreak[st.Quality]; ok {
		return v
	}
	return 0.162
}

// PolishCeilingFor: 這顆料的倍率天花板（種水越好，磨得越高）。
func PolishCeilingFor(st *Stone) float64 {
	if st == nil {
		return 6.0
	}
	if v, ok := polishCeiling[st.Quality]; ok {
		return v
	}
	return 6.0
}

// PolishIdealForce: 這顆料吃得住的力度。種水好＝能吃重壓；裂多＝只能輕手。
//
//	磚頭料 1　豆種/油青 2　冰種/玻璃種 3
//	每條裂紋 −0.5，深裂再 −1
func PolishIdealForce(st *Stone) int {
	if st == nil {
		return PolishForceNormal
	}
	base := 2.0
	switch st.Quality {
	case Brick:
		base = 1
	case Bean, OilGreen:
		base = 2
	case Icy, Glass:
		base = 3
	}
	base -= 0.5 * float64(len(st.CrackCells))
	if st.CracksDeep {
		base--
	}
	f := int(math.Round(base))
	if f < PolishForceLight {
		f = PolishForceLight
	}
	if f > PolishForceHeavy {
		f = PolishForceHeavy
	}
	return f
}

// PolishBreakProb: 這個力度用在這個石頭上，每層的爆裂機率。
// 配對時最低；每差一級 +18%。
func PolishBreakProb(st *Stone, force int, breakModifier float64) float64 {
	mismatch := math.Abs(float64(force - PolishIdealForce(st)))
	p := PolishBaseBreakFor(st) + PolishMismatch*mismatch - breakModifier
	if st != nil {
		p += PolishCrackRisk * float64(len(st.CrackCells))
		if st.CracksDeep {
			p += PolishDeepRisk
		}
	}
	if p < 0.02 {
		p = 0.02
	}
	if p > 0.95 {
		p = 0.95
	}
	return p
}

// PolishMatched: 力度是否配得上這顆料。
func PolishMatched(st *Stone, force int) bool {
	return force == PolishIdealForce(st)
}

// PolishMultiplier: 爬到第 stage 層的倍率（受這顆料的種水天花板限制）。
func PolishMultiplier(st *Stone, stage int) float64 {
	m := PolishStartMult * math.Pow(PolishStepGain, float64(stage))
	if cap := PolishCeilingFor(st); m > cap {
		m = cap
	}
	return m
}

// PolishState is the live run state (persisted per stone).
type PolishState struct {
	Stage int
	Force int // chosen before starting; cannot change
	Alive bool
}

func NewPolishState(force int) *PolishState {
	return &PolishState{Stage: 0, Force: force, Alive: true}
}

func (p *PolishState) Multiplier(st *Stone) float64 { return PolishMultiplier(st, p.Stage) }

// AtTop: 已到天花板。
func (p *PolishState) AtTop() bool { return p.Stage >= PolishMaxStage }

// Advance rolls one more layer. Returns (alive, brokeAtStage).
func (p *PolishState) Advance(st *Stone, r Rand, breakModifier float64) (bool, int) {
	if r.Float64() < PolishBreakProb(st, p.Force, breakModifier) {
		p.Alive = false
		return false, p.Stage + 1
	}
	p.Stage++
	return true, -1
}

// CashPayout: stop and bank.
func (p *PolishState) CashPayout(st *Stone, baseValue int) int {
	return int(float64(baseValue) * PolishMultiplier(st, p.Stage))
}

// PolishBrokenPayout: 磨崩當下能救回的金額（當前倍率 × 35%）。
func PolishBrokenPayout(st *Stone, stage, baseValue int) int {
	return int(float64(baseValue) * PolishMultiplier(st, stage) * PolishBreakSalvage)
}

// PolishForceName: 力度名稱。
func PolishForceName(force int) string {
	switch force {
	case PolishForceLight:
		return "輕磨"
	case PolishForceHeavy:
		return "重磨"
	default:
		return "正磨"
	}
}

// PolishFeel: 機器的手感——誠實反映力度配不配得上這顆料，
// 但不會直接說出種水。玩家要靠這個（和打燈報告）決定繼續還是收手。
func PolishFeel(st *Stone, force int) string {
	if st == nil {
		return "機器聲沉悶，聽不出所以然。"
	}
	mismatch := int(math.Abs(float64(force - PolishIdealForce(st))))
	switch mismatch {
	case 0:
		switch st.Quality {
		case Glass:
			return "機器順暢，石屑細如粉末，磨面透出一層水光——這料吃得下這個力度。"
		case Icy:
			return "機器順暢，石屑細勻，磨面開始起膠——這料吃得下這個力度。"
		case OilGreen:
			return "機器順暢，石屑油潤，色根隨磨面浮出來——這料吃得下這個力度。"
		case Bean:
			return "機器順暢，石屑均勻，磨面見色——這料吃得下這個力度。"
		default:
			return "機器順暢，石屑發白疏鬆——這料吃得下這個力度。"
		}
	case 1:
		switch {
		case force > PolishIdealForce(st):
			return "機器微微跳動，石屑偏粗，磨面出現細小崩口——壓得太重了。"
		default:
			return "機器空轉，磨面幾乎不動，石屑黏膩——力度太輕，磨不動這料。"
		}
	default:
		return "機器劇烈震動，石面接連崩口，聲音刺耳——完全壓不住這顆料。"
	}
}
