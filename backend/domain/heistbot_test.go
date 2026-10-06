package domain

import "testing"

func TestHeistHasEnoughKindBots(t *testing.T) {
	kind := 0
	calm := 0
	for _, b := range HeistBots {
		if b.CoopRate >= 0.95 && b.GreedAt > 1.0 && !b.Revenge {
			kind++
		}
		if b.CoopRate >= 0.85 && b.GreedAt >= 0.90 && !b.Revenge {
			calm++
		}
	}
	if kind < 4 {
		t.Fatalf("善良 bot 至少要 4 隻，現在只有 %d", kind)
	}
	if calm < 10 {
		t.Fatalf("冷靜正常 bot 至少要 10 隻，現在只有 %d", calm)
	}
}
