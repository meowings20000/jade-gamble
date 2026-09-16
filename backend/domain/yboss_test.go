package domain

import (
	"math"
	"testing"
)

func TestYBossCutEV(t *testing.T) {
	r := NewDetRand(3)
	const N = 500_000
	total := 0.0
	labels := map[string]int{}
	for i := 0; i < N; i++ {
		m, label := YBossCut(r)
		total += m
		labels[label]++
	}
	ev := total / N
	if math.Abs(ev-0.95) > 0.01 {
		t.Errorf("YBoss cut EV: %.4f want 0.95", ev)
	} else {
		t.Logf("YBoss cut EV: %.4f (house edge %.2f%%)", ev, (1-ev)*100)
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
