package domain

import (
	"math"
	"testing"
)

// 彩蛋機率與 EV 包絡：5% 出寶石，整體切石 EV 仍 < 1（不破壞 6:4）。
func TestGemEasterEggEnvelope(t *testing.T) {
	r := NewDetRand(20260916)
	const N = 400_000
	hits := 0
	for i := 0; i < N; i++ {
		if _, ok := RollGem(r); ok {
			hits++
		}
	}
	rate := float64(hits) / N
	if math.Abs(rate-GemChance) > 0.004 {
		t.Errorf("彩蛋機率 %.3f%%，應該 %.1f%%", rate*100, GemChance*100)
	}

	// 寶石內部 EV
	gemEV := 0.0
	sum := 0.0
	for _, g := range Gems {
		gemEV += g.Prob * g.Mult
		sum += g.Prob
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("寶石機率合計 %.6f，應該 1", sum)
	}
	// 整體：95% 走正常切石（公斤料 EV ≈0.96）× 5% 彩蛋
	overall := (1-GemChance)*0.96 + GemChance*gemEV
	if overall >= 1.0 {
		t.Errorf("整體切石 EV %.3f ≥ 1，彩蛋給太多", overall)
	}
	if overall < 0.93 {
		t.Errorf("整體切石 EV %.3f，彩蛋後抽太兇", overall)
	}
	t.Logf("彩蛋 %.1f%% · 寶石內部 EV %.3f · 整體 EV %.3f", rate*100, gemEV, overall)
}

// 越稀有倍率越高、機率越低（鑽石最貴最少）。
func TestGemRarityOrder(t *testing.T) {
	for i := 1; i < len(Gems); i++ {
		if Gems[i].Mult <= Gems[i-1].Mult {
			t.Errorf("%s 倍率沒有比 %s 高", Gems[i].Name, Gems[i-1].Name)
		}
		if Gems[i].Prob >= Gems[i-1].Prob {
			t.Errorf("%s 機率沒有比 %s 低", Gems[i].Name, Gems[i-1].Name)
		}
	}
	if Gems[len(Gems)-1].Name != "鑽石" {
		t.Errorf("最稀有應該是鑽石，得到 %s", Gems[len(Gems)-1].Name)
	}
	if _, ok := GemByKey("ruby"); !ok {
		t.Error("找不到 ruby")
	}
	// 彩蛋賠付吃石價（不是真值），雙倍券照樣算
	st := &Stone{Price: 1000, Quality: Bean, Variety: Base}
	ruby, _ := GemByKey("ruby")
	if got := GemPayout(st, ruby, false); got != 1800 {
		t.Errorf("紅寶石賠付 %d，應該 1800", got)
	}
	if got := GemPayout(st, ruby, true); got != 3600 {
		t.Errorf("雙倍券紅寶石賠付 %d，應該 3600", got)
	}
}

// 中了彩蛋就不走玉石結算（賠付由寶石決定）。
func TestCutRevealWithGemPath(t *testing.T) {
	st := &Stone{Price: 2000, Quality: Brick, Variety: Base}
	// 大量抽樣：有中彩蛋的賠付一定是寶石倍率，沒中的一定是 BaseValue
	gotGem := false
	for i := 0; i < 20000; i++ {
		payout, gem := CutRevealWithGem(st, false, NewDetRand(uint64(i)+1))
		if gem != nil {
			gotGem = true
			if payout != int(float64(st.Price)*gem.Mult) {
				t.Fatalf("彩蛋賠付 %d 不等於 石價×%.2f", payout, gem.Mult)
			}
		} else if payout != st.BaseValue() {
			t.Fatalf("非彩蛋賠付 %d 應該等於 BaseValue %d", payout, st.BaseValue())
		}
	}
	if !gotGem {
		t.Error("兩萬次都沒中彩蛋")
	}
}
