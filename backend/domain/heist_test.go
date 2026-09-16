package domain

import "testing"

func seats(n, entry int) []HeistSeat {
	out := []HeistSeat{}
	for i := 1; i <= n; i++ {
		out = append(out, HeistSeat{UserID: i, Name: string(rune('A' + i - 1)), Alive: true, Entry: entry})
	}
	return out
}

func coop(s []HeistSeat, ids ...int) {
	for i := range s {
		for _, id := range ids {
			if s[i].UserID == id {
				s[i].Action = HeistCooperate
			}
		}
	}
}

func betray(s []HeistSeat, from, target int) {
	for i := range s {
		if s[i].UserID == from {
			s[i].Action = HeistBetray
			s[i].Target = target
		}
	}
}

// 四人互相合作：+6 格，沒人死。
// killRand：測試用——Float64 永遠回 0（= 0 < 0.70 → 刺殺必成功），
// 讓「背叛殺人」的測試不受新的 70% 機率影響。
func init() {
	// 測試固定成「刺殺必成功」，避免 60/20/20 機率讓斷言隨機翻車。
	HeistKillRoll = func(Rand) int { return 90 }
}

type killRand struct{}

func (killRand) Float64() float64 { return 0.9 }
func (killRand) Intn(n int) int   { return 0 }

func TestHeistMutualCoop(t *testing.T) {
	s := seats(4, 5000)
	coop(s, 1, 2, 3, 4)
	r := ResolveHeistRound(s, 0, HeistTargetBase, killRand{})
	if r.Progress != 6 || r.MutualPair != 6 {
		t.Errorf("四人互相合作應該 +6 格，得到 progress=%d pair=%d", r.Progress, r.MutualPair)
	}
	if len(r.Deaths) != 0 {
		t.Errorf("沒人背叛不該有人死: %v", r.Deaths)
	}
}

// 我背叛你、你合作 → 你死、我拿你 70% 入場費，而且你知道是我。
func TestHeistBetrayKillsCooperating(t *testing.T) {
	s := seats(4, 5000)
	coop(s, 2, 3, 4)
	betray(s, 1, 4)
	r := ResolveHeistRound(s, 0, HeistTargetBase, killRand{})
	if r.Deaths[4] != 1 {
		t.Fatalf("4 號應該被 1 號殺，得到 %v", r.Deaths)
	}
	if r.Looters[1] != 3500 {
		t.Errorf("兇手應該拿 5000×70%%=3500，得到 %d", r.Looters[1])
	}
	if r.Exposed[4] != 1 {
		t.Errorf("4 號應該知道是 1 號動的手，得到 %v", r.Exposed[4])
	}
	if r.Progress != 3 {
		t.Errorf("剩下三人互相合作應該 +3，得到 %d", r.Progress)
	}
}

// 互投背叛 → 互相抵銷，兩個都沒死。
func TestHeistMutualBetrayalNoDeath(t *testing.T) {
	s := seats(4, 5000)
	coop(s, 3, 4)
	betray(s, 1, 2)
	betray(s, 2, 1)
	r := ResolveHeistRound(s, 0, HeistTargetBase, killRand{})
	if len(r.Deaths) != 0 {
		t.Errorf("互相背叛應該兩個都沒死: %v", r.Deaths)
	}
	if r.Collapse {
		t.Error("只有兩人背叛，不該崩塌")
	}
}

// 全員同時背叛 → 崩塌：全部死亡、獎池沒收（這就是「互相殘殺全部死掉」）。
func TestHeistCollapseAllDead(t *testing.T) {
	s := seats(4, 5000)
	for i := range s {
		s[i].Action = HeistBetray
		s[i].Target = (s[i].UserID % 4) + 1
	}
	r := ResolveHeistRound(s, 12, HeistTargetBase, killRand{})
	if !r.Collapse {
		t.Fatal("全員背叛應該崩塌")
	}
	if len(r.Deaths) != 4 {
		t.Errorf("崩塌應該四人全死，得到 %v", r.Deaths)
	}
	if len(r.Looters) != 0 {
		t.Errorf("崩塌沒收，不該有人拿到錢: %v", r.Looters)
	}
}

// 三人循環背叛（A→B、B→C、C→A）＋一人合作 → 三人全死、合作那位獨活。
func TestHeistRingWipeout(t *testing.T) {
	s := seats(4, 5000)
	coop(s, 4)
	betray(s, 1, 2)
	betray(s, 2, 3)
	betray(s, 3, 1)
	r := ResolveHeistRound(s, 6, HeistTargetBase, killRand{})
	if len(r.Deaths) != 3 {
		t.Errorf("循環背叛應該死三人，得到 %v", r.Deaths)
	}
	if _, dead := r.Deaths[4]; dead {
		t.Error("合作的那位不該死")
	}
	if r.Progress != 6 {
		t.Errorf("只有一人合作沒有配對，進度應該維持 6，得到 %d", r.Progress)
	}
}

// 獎池：4 人分 = 1.5× 入場費；獨吞 = 6× 入場費。
func TestHeistPayout(t *testing.T) {
	for _, g := range []ShopGrade{KiloGrade, FeatureGrade, WindowGrade} {
		pot := HeistPot(g)
		if pot != HeistEntry(g)*6 {
			t.Errorf("grade %d 獎池應該 6× 入場費 %d，得到 %d", g, HeistEntry(g)*6, pot)
		}
		p4 := HeistPayout(g, pot, []int{1, 2, 3, 4})
		if p4[1] != HeistEntry(g)*3/2 {
			t.Errorf("四人平分應該各 1.5× 入場費 %d，得到 %d", HeistEntry(g)*3/2, p4[1])
		}
		p1 := HeistPayout(g, pot, []int{2})
		if p1[2] != pot {
			t.Errorf("獨吞應該全拿 %d，得到 %d", pot, p1[2])
		}
	}
}
