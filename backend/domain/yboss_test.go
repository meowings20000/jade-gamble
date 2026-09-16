package domain

import (
	"math"
	"testing"
)

// Y佬切一刀的賠率表本身要自洽（機率合計 1、骰出來的結果一定在表裡）。
// 經濟層面的驗證在 balance_test.go 的 TestYBossCutOdds（勝率/EV）。
func TestYBossCutTableIntegrity(t *testing.T) {
	odds := YBossCutOdds()
	sum := 0.0
	inTable := map[string]float64{}
	for _, o := range odds {
		p := o["prob"].(float64)
		sum += p
		inTable[o["label"].(string)] = o["mult"].(float64)
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("賠率表機率合計 %.6f，應該 1", sum)
	}
	if len(odds) != 6 {
		t.Errorf("賠率表應該 6 格，得到 %d", len(odds))
	}
	// 骰 20 萬次，每個結果都必須是表裡的其中一格
	r := NewDetRand(7)
	for i := 0; i < 200_000; i++ {
		m, label := YBossCut(r)
		want, ok := inTable[label]
		if !ok {
			t.Fatalf("骰出表外的結果: %q", label)
		}
		if m != want {
			t.Fatalf("%s 倍率 %.2f 與表不符（表裡是 %.2f）", label, m, want)
		}
	}
	// 前端顯示用的表要有最高倍那格（×15 帝王綠）——這是 Y佬的招牌
	maxMult := 0.0
	for _, v := range inTable {
		if v > maxMult {
			maxMult = v
		}
	}
	if maxMult != 15.0 {
		t.Errorf("最高倍率 %.1f，Y佬模式應該是 ×15 帝王綠", maxMult)
	}
}

func TestYBossPolishSteps(t *testing.T) {
	for i := 0; i < len(YBossPolishLadder)-1; i++ {
		ev := YBossPolishStepEV(i)
		if math.Abs(ev-0.95) > 0.01 {
			t.Errorf("rung %d step EV: %.4f want 0.95", i, ev)
		}
	}
	// full ladder reach rate + overall EV (cash-at-top policy)
	r := NewDetRand(9)
	const N = 100_000
	top := 0
	total := 0.0
	for i := 0; i < N; i++ {
		rung := 0
		alive := true
		for alive && rung < len(YBossPolishLadder)-1 {
			rung, alive = YBossPolishAdvance(r, rung)
		}
		if rung == len(YBossPolishLadder)-1 && alive {
			top++
		}
		if alive {
			total += YBossPolishLadder[rung].Mult
		}
	}
	t.Logf("polish: top reach %.2f%%, cash-at-top EV %.3f×stake", float64(top)/N*100, total/N)
}
