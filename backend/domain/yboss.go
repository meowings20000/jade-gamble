package domain

import "errors"

// YBoss mode (Y佬模式): pure stake gambling — no stone management.
//
// The player posts funds and picks ONE of two games:
//   - cut (切一刀): one-shot reveal, payout = stake × roll, EV ≈ 0.95
//   - polish ladder (磨石): 6 rungs; each rung multiplies the stake and
//     raises the rarity band; survival chance drops per rung. Crash = 0.
//
// Every advance is one atomic server-side wager; the only persisted state
// is the session's current rung.

var ErrYBossChoice = errors.New("要選切或磨")

const (
	YBossMinStake = 100
	YBossMaxStake = 1_000_000
)

// yBossCutRolls: Y佬模式「切一刀」的賠率表——這一組是 Y佬專用，
// 刻意保留賭性（58% 歸零、1% 帝王綠 ×15），不套用主玩法的 6:4 校準。
// 玩家看到的就是這張表，賠率與顯示一致。
// EV = .18*1 + .12*1.5 + .07*3 + .04*5 + .01*15 = 0.92
// (TestYBossCutEV 鎖住這組數字。)
var yBossCutRolls = []struct {
	prob  float64
	mult  float64
	label string
}{
	{0.58, 0.0, "磚頭料"},
	{0.18, 1.0, "豆種"},
	{0.12, 1.5, "糯種"},
	{0.07, 3.0, "冰種"},
	{0.04, 5.0, "玻璃種"},
	{0.01, 15.0, "帝王綠"},
}

// YBossCutOdds: 給前端顯示的賠率表（單一真相，避免前後端各寫一份）。
func YBossCutOdds() []map[string]any {
	out := []map[string]any{}
	for _, e := range yBossCutRolls {
		out = append(out, map[string]any{"label": e.label, "prob": e.prob, "mult": e.mult})
	}
	return out
}

// YBossCut resolves one cut bet. Returns (multiplier, label).
func YBossCut(r Rand) (float64, string) {
	p := r.Float64()
	acc := 0.0
	for _, e := range yBossCutRolls {
		acc += e.prob
		if p < acc {
			return e.mult, e.label
		}
	}
	last := yBossCutRolls[len(yBossCutRolls)-1]
	return last.mult, last.label
}

// YBossRung: one step of the polish ladder.
type YBossRung struct {
	Mult    float64
	Survive float64 // chance of surviving the advance TO this rung
	Label   string
}

// YBossPolishLadder: per-step EV = Mult[i+1]*Survive[i+1]/Mult[i] ≈ 0.95
// (TestYBossPolishSteps locks this). Top rung ×7.5, reached ~9% of runs.
var YBossPolishLadder = []YBossRung{
	{Mult: 1.0, Survive: 1.00, Label: "原石"},
	{Mult: 1.3, Survive: 0.73, Label: "粗磨見色"},
	{Mult: 1.75, Survive: 0.70, Label: "細磨起膠"},
	{Mult: 2.4, Survive: 0.69, Label: "拋光出熒"},
	{Mult: 3.4, Survive: 0.67, Label: "冰種初成"},
	{Mult: 5.0, Survive: 0.65, Label: "高冰起熒"},
	{Mult: 7.5, Survive: 0.63, Label: "玻璃種極品"},
}

// YBossPolishAdvance: attempt rung → rung+1.
// Returns (newRung, alive). !alive means the bet is lost (payout 0).
func YBossPolishAdvance(r Rand, rung int) (int, bool) {
	if rung >= len(YBossPolishLadder)-1 {
		return rung, true // already at top
	}
	next := YBossPolishLadder[rung+1]
	if r.Float64() < next.Survive {
		return rung + 1, true
	}
	return rung + 1, false // broke at rung+1
}

// YBossPolishPayout: cash out at rung.
func YBossPolishPayout(stake int, rung int) int {
	if rung < 0 || rung >= len(YBossPolishLadder) {
		return 0
	}
	return int(float64(stake) * YBossPolishLadder[rung].Mult)
}

// YBossPolishStepEV: per-step EV of advancing from rung i.
func YBossPolishStepEV(i int) float64 {
	if i < 0 || i >= len(YBossPolishLadder)-1 {
		return 1
	}
	return YBossPolishLadder[i+1].Mult * YBossPolishLadder[i+1].Survive / YBossPolishLadder[i].Mult
}
