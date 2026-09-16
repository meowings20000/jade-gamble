package domain

// 隱藏彩蛋（2026-09-16）：切石偶爾切出「不是玉石」的寶石。
//
// 純驚喜設計：
//   - 只有切石會中（刮／磨不會），機率 5%，任何檔位都一樣。
//   - 中了就整顆走寶石那把尺：賠率＝石價 × 倍率，不走種水／異色。
//   - 打燈看不到（報告講的是玉石），所以真的是「切下去才知道」。
//
// 經濟：寶石內部 EV = 1.13，整體切石 EV = 0.95×0.98 + 0.05×1.13 ≈ 0.988 < 1，
// 主玩法的 6:4 不會被彩蛋破壞（TestGemEasterEggEnvelope 鎖住）。
const GemChance = 0.05

// Gem: 一顆寶石彩蛋。
type Gem struct {
	Key  string
	Name string
	Mult float64
	Prob float64 // 在「已中彩蛋」的前提下
}

// Gems: 越稀有越貴（鑽石 1%）。
var Gems = []Gem{
	{Key: "amethyst", Name: "紫水晶", Mult: 0.70, Prob: 0.38},
	{Key: "topaz", Name: "黃玉", Mult: 1.00, Prob: 0.27},
	{Key: "sapphire", Name: "藍寶石", Mult: 1.30, Prob: 0.20},
	{Key: "ruby", Name: "紅寶石", Mult: 1.80, Prob: 0.10},
	{Key: "emerald", Name: "祖母綠", Mult: 2.60, Prob: 0.04},
	{Key: "diamond", Name: "鑽石", Mult: 5.00, Prob: 0.01},
}

// RollGem: 擲彩蛋。沒中就 (Gem{}, false)。
func RollGem(r Rand) (Gem, bool) {
	if r.Float64() >= GemChance {
		return Gem{}, false
	}
	p := r.Float64()
	acc := 0.0
	for _, g := range Gems {
		acc += g.Prob
		if p < acc {
			return g, true
		}
	}
	return Gems[len(Gems)-1], true
}

// GemByKey: 依 key 取寶石（前端渲染／紀錄用）。
func GemByKey(key string) (Gem, bool) {
	for _, g := range Gems {
		if g.Key == key {
			return g, true
		}
	}
	return Gem{}, false
}

// GemPayout: 彩蛋賠付＝石價 × 倍率（雙倍券照樣算）。
func GemPayout(s *Stone, g Gem, doubleCoupon bool) int {
	p := int(float64(s.Price) * g.Mult)
	if doubleCoupon {
		p *= 2
	}
	return p
}

// GemOdds: 給前端／文件用的機率表（整體機率＝GemChance × Prob）。
func GemOdds() []map[string]any {
	out := []map[string]any{}
	for _, g := range Gems {
		out = append(out, map[string]any{
			"key": g.Key, "name": g.Name, "mult": g.Mult,
			"prob": g.Prob, "overall": g.Prob * GemChance,
		})
	}
	return out
}
