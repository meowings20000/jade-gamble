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
	s.CracksDeep = r.Float64() < 0.18

	s.LightHint = buildLightHint(s, r)
	return s
}

// rollQuality: distribution tuned so EV/price lands near GradeEcon.EVMult.
// See econ_test.go for the assertion. Solved jointly with the crack penalty
// (0.919) and variety boost (~1.17) so each grade hits its target edge.
func rollQuality(grade ShopGrade, r Rand) Quality {
	p := r.Float64()
	switch grade {
	case KiloGrade: // heavy tail down, tiny glass chance
		switch {
		case p < 0.08:
			return Brick
		case p < 0.40:
			return Bean
		case p < 0.82:
			return OilGreen
		case p < 0.99:
			return Icy
		default:
			return Glass
		}
	case FeatureGrade:
		switch {
		case p < 0.06:
			return Brick
		case p < 0.38:
			return Bean
		case p < 0.86:
			return OilGreen
		case p < 0.985:
			return Icy
		default:
			return Glass
		}
	default: // WindowGrade
		switch {
		case p < 0.05:
			return Brick
		case p < 0.38:
			return Bean
		case p < 0.88:
			return OilGreen
		case p < 0.985:
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

func buildLightHint(s *Stone, r Rand) string {
	// Honest core, lies mixed in at LightLieRate.
	truth := s.Quality != Brick && !s.CracksDeep
	if r.Float64() < s.LightLieRate {
		truth = !truth
	}
	if s.Egg == "b_fake" {
		truth = false // the lie is baked into the egg
	}
	style := r.Intn(3)
	if truth {
		return []string{
			fmt.Sprintf("打灯光圈%s，水头%s，皮壳砂发细腻，是块值得认真对待的料。", brightWord(s, r), waterWord(s, r)),
			fmt.Sprintf("光能吃进去，色根沉稳，%s。行家会多看两眼。", feelWord(s)),
			fmt.Sprintf("透光均匀，%s，裂纹走向避得开。", toneWord(s)),
		}[style]
	}
	return []string{
		fmt.Sprintf("打燈光線散在表面，进不去，水头短。%s", feelWord(s)),
		fmt.Sprintf("光圈浑浊，%s，砂发粗糙翻手。", toneWord(s)),
		"光在皮壳表面乱跳，看不出名堂，这料子玄。",
	}[style]
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
