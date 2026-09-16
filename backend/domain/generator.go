package domain

import (
	"fmt"
	"math"
)

// GenerateStone rolls a complete sealed stone for a shop shelf slot.
// Economics: the EV table lives in quality.go / GradeEcon; generator.go
// owns the quality distribution so tests can pin exact EV per grade.
//
// Bianhe egg: KiloGrade only, 0.01% chance, forces Glass + ImperialGreen.
// B-fake egg: FeatureGrade only, 0.3%, perfect-looking hint, actual Brick.
func GenerateStone(grade ShopGrade, r Rand) *Stone {
	s := &Stone{
		ID:    newStoneID(r),
		Grade: grade,
		Seed:  uint64(r.Intn(math.MaxInt32))<<32 | uint64(r.Intn(math.MaxInt32)),
	}
	s.Price = Range(r, grade.Econ().PriceLo, grade.Econ().PriceHi)
	s.LightLieRate = grade.Econ().LieRate

	// Eggs.
	switch grade {
	case KiloGrade:
		if r.Float64() < 0.0001 {
			s.Egg = "bianhe"
			s.BianheGuaranteed = true
			s.Quality, s.Variety = Glass, ImperialGreen
			s.LightHint = "皮壳粗糠如砂纸，丑得不像话。老师傅路过看了一眼，摇了摇头。"
			return s
		}
	case FeatureGrade:
		if r.Float64() < 0.003 {
			s.Egg = "b_fake"
			s.Quality, s.Variety = Brick, Base
			s.LightHint = "松花鲜艳成线，蟒带隆起明显，打灯光圈清澈见底——皮壳表现堪称完美。"
			return s
		}
	}

	s.Quality = rollQuality(grade, r)
	if s.Quality == Brick {
		s.Variety = Base
	} else {
		s.Variety = rollColorVariety(r)
	}
	if s.Variety == InkGreen {
		s.InkHidden = true
	}

	// Cracks: independent of quality; deep crack rarer on higher grades.
	nCracks := 0
	switch grade {
	case KiloGrade:
		nCracks = Range(r, 0, 4)
	case FeatureGrade:
		nCracks = Range(r, 0, 3)
	case WindowGrade:
		nCracks = Range(r, 0, 2)
	}
	s.CrackCells = make([]int, 0, nCracks)
	seen := map[int]bool{}
	for i := 0; i < nCracks; i++ {
		for tries := 0; tries < 8; tries++ {
			c := r.Intn(96)
			if !seen[c] {
				seen[c] = true
				s.CrackCells = append(s.CrackCells, c)
				break
			}
		}
	}
	s.CracksDeep = r.Float64() < 0.06 // 深裂要罕見，否則會把「贏」打成「輸」

	s.LightHint = buildLightHint(s, r)
	return s
}

// rollQuality: distribution tuned so EV/price lands near GradeEcon.EVMult.
// See econ_test.go for the assertion. Solved jointly with the crack penalty
// (0.919) and variety boost (~1.17) so each grade hits its target edge.
func rollQuality(grade ShopGrade, r Rand) Quality {
	p := r.Float64()
	switch grade {
	case KiloGrade:
		// 蒙頭料：便宜、賠率最大——磚頭多，但冰種/玻璃種的比例也最高（4% 玻璃）。
		switch {
		case p < 0.14:
			return Brick
		case p < 0.42:
			return Bean
		case p < 0.80:
			return OilGreen
		case p < 0.96:
			return Icy
		default:
			return Glass
		}
	case FeatureGrade:
		switch {
		case p < 0.05:
			return Brick
		case p < 0.40:
			return Bean
		case p < 0.87:
			return OilGreen
		case p < 0.98:
			return Icy
		default:
			return Glass
		}
	default: // WindowGrade
		// 開窗料：穩定、賠率小——幾乎沒有磚頭，油青種（小賺）佔一半以上。
		switch {
		case p < 0.02:
			return Brick
		case p < 0.38:
			return Bean
		case p < 0.90:
			return OilGreen
		case p < 0.99:
			return Icy
		default:
			return Glass
		}
	}
}

func rollVarietyName(r Rand) string { _ = r; return "" }

// rollColorVariety picks a color variety (called for Bean+).
func rollColorVariety(r Rand) ColorVariety {
	p := r.Float64()
	acc := 0.0
	for _, v := range []ColorVariety{Violet, BlueWater, WhiteGreen, FloatingBlue, InkGreen, YellowGreen, SpringPurple, FortuneThree, ImperialGreen} {
		acc += varietyProb[v]
		if p < acc {
			return v
		}
	}
	return Base
}

// BaseValue: the "true" market value of the stone's interior, before
// processing decisions (cut/scratch/polish).
func (s *Stone) BaseValue() int {
	v := float64(s.Price) * s.Quality.Multiplier() * s.Variety.Multiplier()
	// Deep crack slashes interior value (套型 can partially save it).
	if s.CracksDeep {
		v *= 0.55
	}
	return int(v)
}

// buildWindowHint: 開窗料看得到肉，所以打燈直接報種水與裂紋——
// 但仍有 LightLieRate 的機率報錯（可能謊報種水，或把裂紋數量說錯）。
func buildWindowHint(s *Stone, r Rand) string {
	tier := s.Quality
	if r.Float64() < s.LightLieRate {
		// 謊報：往上或往下報一級
		if r.Float64() < 0.5 && tier > Brick {
			tier--
		} else if tier < Glass {
			tier++
		}
	}
	water := map[Quality]string{
		Brick: "肉質發乾，沒什麼水頭", Bean: "水頭短，色偏淡", OilGreen: "油潤見色，水頭中等",
		Icy: "水頭足，透光見底", Glass: "螢光隱隱，透得發亮",
	}[tier]
	cracks := len(s.CrackCells)
	deep := s.CracksDeep
	if r.Float64() < s.LightLieRate {
		if cracks > 0 {
			cracks--
		} else {
			cracks++
		}
		if deep && r.Float64() < 0.5 {
			deep = false
		}
	}
	crackTxt := "窗邊無明顯裂紋"
	switch {
	case cracks <= 0:
		crackTxt = "窗邊無明顯裂紋"
	case deep:
		crackTxt = fmt.Sprintf("窗下見 %d 道裂，其中一道較深，切記避開", cracks)
	default:
		crackTxt = fmt.Sprintf("側面見 %d 道細裂，走向避得開", cracks)
	}
	return fmt.Sprintf("【開窗】開窗處見%s底，%s；%s。", tier.Name(), water, crackTxt)
}

// 打燈（付費）才看得到的描述。三檔的資訊量刻意不同：
//
//	蒙頭料（公斤料）：模糊，只講感覺，沒有結論——線索藏在字裡行間。
//	表現料：中等，會提到表現特徵（松花、蟒帶、砂皮），方向隱約可見。
//	開窗料：明確，直接報種水與裂紋（窗已經開了，本來就看得見）。
//
// DescribeFree: 沒打燈時「肉眼可見」的描述——資訊量依檔位遞減的設計：
//
//	蒙頭料（公斤料）：什麼都沒有（本來就是全盲的）。
//	表現料：皮殼上的表現，但看不出深淺。
//	開窗料：窗面擦出來的一片色，同樣看不出準確種水。
//
// 想要準確的資訊就得付費打燈（見 buildLightHint 三檔不同的資訊量）。
func DescribeFree(s *Stone) string {
	switch s.Grade {
	case WindowGrade:
		if s.InkHidden {
			return "窗面死黑，光都進不去——像塊廢料。"
		}
		return map[Quality]string{
			Glass:    "窗面擦出一片淺綠，通透得刺眼，但看不準水頭深淺。",
			Icy:      "窗面透出一線光，肉質看著細，深淺說不準。",
			OilGreen: "窗面擦出油亮的一片色，濃淡難辨。",
			Bean:     "窗面見淡色，顆粒粗細看不清楚。",
		}[s.QualityMax()]
	case FeatureGrade:
		if s.InkHidden {
			return "黑皮一絲表現都沒有，看不出名堂。"
		}
		return "皮殼上見鬆散的表現，走向斷斷續續，看不出深淺。"
	default:
		return ""
	}
}

// DescribeFreeByGrade: 給「看不到實物」的場合（例如市場掛單）用的模糊描述。
func DescribeFreeByGrade(g ShopGrade) string {
	switch g {
	case WindowGrade:
		return "賣家說擦出過一片色，深淺不好說。"
	case FeatureGrade:
		return "皮上有點表現，斷斷續續。"
	default:
		return "蒙頭料，什麼都看不出來。"
	}
}

// QualityMax: 這顆石頭的種水（不同檔位都可能出現異色）。
func (s *Stone) QualityMax() Quality { return s.Quality }

func buildLightHint(s *Stone, r Rand) string {
	if s.Grade == WindowGrade {
		return buildWindowHint(s, r)
	}
	if s.Grade == FeatureGrade {
		return buildFeatureHint(s, r)
	}
	return buildBlindHint(s, r)
}

// buildBlindHint: 蒙頭料。打燈只看到一團光暈，說不出所以然——
// 好料的光暈「收」、「緊」，差料「散」、「浮」；謊報率 LightLieRate。
func buildBlindHint(s *Stone, r Rand) string {
	good := s.Quality != Brick && !s.CracksDeep
	if s.Egg == "b_fake" {
		good = false
	}
	if r.Float64() < s.LightLieRate {
		good = !good
	}
	wrap := []string{
		"皮殼砂粒粗細不一，摸不出所以然。光打下去%s，水頭看不透——這種料，全憑下刀那一刻。",
		"轉了兩圈燈，%s。皮上沒什麼表現，說不準，得看切口。",
		"燈下%s，其餘什麼都沒看出來。蒙頭就是蒙頭。",
	}
	goodWord := []string{"光暈收得緊、散得慢", "有那麼一瞬間光像是被吃住了", "光圈邊緣沉下去一線"}
	badWord := []string{"光整個浮在皮上，散得很快", "光圈渾濁，進不去", "光在表面亂跳，一點都不收"}
	pick := func(list []string) string { return list[r.Intn(len(list))] }
	word := pick(badWord)
	if good {
		word = pick(goodWord)
	}
	return fmt.Sprintf(wrap[r.Intn(len(wrap))], word)
}

// buildFeatureHint: 表現料。皮殼上有表現，看得出一點方向，
// 但真正的種水還是要開了才知道（謊報率同 LightLieRate）。
func buildFeatureHint(s *Stone, r Rand) string {
	good := s.Quality != Brick && !s.CracksDeep
	if s.Egg == "b_fake" {
		good = false
	}
	lie := r.Float64() < s.LightLieRate
	if lie {
		good = !good
	}
	manif := []string{"一條鬆散的松花帶", "隱約的蟒帶走勢", "幾點淡淡的白霧"}
	if good {
		manif = []string{"松花帶順著砂皮走，色根看著沉", "蟒帶收得順，砂皮翻得細", "點狀白霧分布均勻"}
	}
	verdict := "表現平平，賭性大"
	if good {
		verdict = "表現尚可，值得留意"
	}
	if lie {
		verdict = "看不出所以然，別太上心"
	}
	return fmt.Sprintf("皮上見%s。%s。", manif[r.Intn(len(manif))], verdict)
}

func brightWord(s *Stone, r Rand) string {
	if s.Variety == Violet && r.Float64() < 0.7 {
		return "泛着淡淡紫晕"
	}
	if s.Variety == InkGreen {
		return "边缘透出一圈幽绿"
	}
	return "清澈见底"
}
func waterWord(s *Stone, _ Rand) string {
	switch s.Quality {
	case Glass:
		return "极长（玻璃种手感）"
	case Icy:
		return "较长"
	default:
		return "中等"
	}
}
func feelWord(s *Stone) string {
	if s.InkHidden {
		return "手感死沉，像块黑炭——但别急着下结论"
	}
	return "手感细腻"
}
func toneWord(s *Stone) string {
	if s.Quality == Brick {
		return "色调死白"
	}
	return "色调正"
}

func newStoneID(r Rand) string {
	const hexdigits = "0123456789abcdef"
	b := make([]byte, 12)
	for i := range b {
		b[i] = hexdigits[r.Intn(16)]
	}
	return "S" + string(b)
}
