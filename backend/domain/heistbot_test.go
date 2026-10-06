package domain

import "testing"

func TestHeistHasEnoughKindBots(t *testing.T) {
	kind := 0
	for _, b := range HeistBots {
		if b.CoopRate >= 0.95 && b.GreedAt > 1.0 && !b.Revenge {
			kind++
		}
	}
	if kind < 4 {
		t.Fatalf("善良 bot 至少要 4 隻，現在只有 %d", kind)
	}
}
