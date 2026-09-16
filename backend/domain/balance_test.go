package domain

import (
	"math"
	"strings"
	"testing"
)

// 三檔定位（2026-09-16）: 蒙頭便宜、賠率最大；開窗穩定、賠率小。
// 這一組數字是平衡的核心，改動任何一個都會動搖玩家體感。
func TestGradeIdentity(t *testing.T) {
	r := NewDetRand(99)
	const N = 120_000
	type stat struct {
		win, ev, sd float64
		lo, hi      int
	}
	measure := func(g ShopGrade) stat {
		var s stat
		var sum, sq, wins float64
		s.lo, s.hi = math.MaxInt32, 0
		for i := 0; i < N; i++ {
			st := GenerateStone(g, r)
			v := float64(st.BaseValue()) / float64(st.Price)
			sum += v
			sq += v * v
			if v >= 1.0 {
				wins++
			}
			if st.Price < s.lo {
				s.lo = st.Price
			}
			if st.Price > s.hi {
				s.hi = st.Price
			}
		}
		s.ev = sum / N
		s.win = wins / N
		s.sd = math.Sqrt(sq/N - s.ev*s.ev)
		return s
	}
	kilo := measure(KiloGrade)
	feat := measure(FeatureGrade)
	win := measure(WindowGrade)

	// 期望值必須略低於 1（莊家有邊際）但不能太狠
	for name, s := range map[string]stat{"公斤料": kilo, "表現料": feat, "開窗料": win} {
		if s.ev > 1.0 {
			t.Errorf("%s EV %.3f > 1（玩家套利）", name, s.ev)
		}
		if s.ev < 0.93 {
			t.Errorf("%s EV %.3f 太低（莊家抽太兇）", name, s.ev)
		}
		// 6:4 開 —— 勝率要落在 55%~65%
		if s.win < 0.55 || s.win > 0.65 {
			t.Errorf("%s 勝率 %.1f%%，應該在 6:4 附近", name, s.win*100)
		}
	}

	// 蒙頭的賠率最大（變異數最高），開窗最穩
	if !(kilo.sd > feat.sd && feat.sd > win.sd) {
		t.Errorf("變異數排序錯: 公斤 %.3f / 表現 %.3f / 開窗 %.3f", kilo.sd, feat.sd, win.sd)
	}
	// 蒙頭比較便宜
	if kilo.lo >= feat.lo {
		t.Errorf("蒙頭料應該最便宜: kilo %d~%d, feature %d~%d", kilo.lo, kilo.hi, feat.lo, feat.hi)
	}
	// 開窗的爆頭（玻璃種）機率最低
	if win.hi < feat.hi {
		t.Logf("開窗價格上限 %d，表現料 %d", win.hi, feat.hi)
	}
}

// 開窗料的打燈必須明確（看得到肉）：要報種水與裂紋。
// 蒙頭料只能是玄的。
func TestWindowHintIsExplicit(t *testing.T) {
	r := NewDetRand(7)
	names := []string{"砖头料", "豆种", "油青种", "冰种", "玻璃种"}
	sawTier := map[string]bool{}
	for i := 0; i < 3000; i++ {
		st := GenerateStone(WindowGrade, r)
		h := st.LightHint
		if !strings.Contains(h, "【開窗】") {
			t.Fatalf("開窗提示缺少標記: %q", h)
		}
		found := false
		for _, n := range names {
			if strings.Contains(h, n) {
				found = true
				sawTier[n] = true
			}
		}
		if !found {
			t.Fatalf("開窗提示沒有報種水: %q", h)
		}
		if !strings.Contains(h, "裂") && !strings.Contains(h, "無明顯裂紋") {
			t.Fatalf("開窗提示沒有報裂紋: %q", h)
		}
	}
	if len(sawTier) < 3 {
		t.Errorf("開窗提示報的種水種類太少: %v", sawTier)
	}
	// 蒙頭料不該出現這個標記
	for i := 0; i < 2000; i++ {
		st := GenerateStone(KiloGrade, r)
		if strings.Contains(st.LightHint, "【開窗】") {
			t.Fatalf("蒙頭料不該有開窗提示: %q", st.LightHint)
		}
	}
}

// Y佬模式切一刀：Y佬專用賠率（58% 歸零、1% ×15），不是主玩法的 6:4。
// 勝率 ≈ 42%、EV ≈ 0.92。
func TestYBossCutOdds(t *testing.T) {
	r := NewDetRand(11)
	const N = 400_000
	wins := 0
	total := 0.0
	for i := 0; i < N; i++ {
		m, _ := YBossCut(r)
		total += m
		if m >= 1.0 {
			wins++
		}
	}
	win := float64(wins) / N
	// 賠率表：豆種 18% + 糯種 12% + 冰種 7% + 玻璃種 4% + 帝王綠 1% = 42% 不虧
	if win < 0.40 || win > 0.44 {
		t.Errorf("Y佬切一刀勝率 %.1f%%，應該 ≈42%%（這組是 Y佬專用賠率）", win*100)
	}
	ev := total / N
	if ev > 1.0 {
		t.Errorf("Y佬切一刀 EV %.3f > 1", ev)
	}
	if ev < 0.90 || ev > 0.94 {
		t.Errorf("Y佬切一刀 EV %.3f，應該 ≈0.92", ev)
	}
	t.Logf("Y佬切一刀: 勝率 %.1f%% EV %.3f", win*100, ev)
}
