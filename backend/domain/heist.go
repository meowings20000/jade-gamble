package domain

import "fmt"

// 奪寶（Heist）——四人囚徒困境（2026-09-16 定案：投票式）
//
// 一桌 4 人，各付入場費（稀有度越高越貴），**互相合作**才推進度：
//
//	合作：和同樣選合作的每個人形成一對 → 每對 +1 格（四人全合作 = 6 格/輪）
//	背叛：這一輪對「一個」對手動手（一輪一張票）
//	      對方沒背叛你 → 他死，你拿他 70% 入場費
//	      對方也背叛你 → 互相抵銷，兩個都不死
//
// 三個結局：
//	全員合作挖到（進度 18）→ 倖存者平分獎池（各 1.5× 入場費）
//	只剩一人 → 獨吞（6× 入場費）
//	**崩塌**：該輪每個活人都投了背叛 → 礦坑崩塌，全部死亡、獎池沒收
//	         （或 5 輪用完還沒挖到 → 入場費沒收）
//
// 循環背叛（A→B、B→C、C→D、D→A）沒有互相抵銷 → 四人全滅，這是「互相殘殺全部死掉」的結局。

const (
	HeistSeats      = 4
	HeistRounds     = 5  // 回合上限（用戶指定 5 輪）
	HeistTargetBase = 18 // 進度目標：四人全合作 3 輪挖到，殺了人就要拖到最後
	HeistKillTake   = 0.70
	HeistPotMul     = 1.5
	// HeistBotWaitSec: 沒真人時，等幾秒才自動補 bot（用戶指定 5 分鐘）
	HeistBotWaitSec = 300
	// HeistRoundSec: 一輪幾秒內要出手，逾時自動算「合作」（用戶指定 30 秒）
	HeistRoundSec = 30 // 獎池 = 4 × 入場費 × 1.5（每人分 1.5× 入場費）
)

const (
	HeistCooperate = "cooperate"
	HeistBetray    = "betray"
)

// HeistEntry: 依稀有度（檔位）的入場費。
func HeistEntry(grade ShopGrade) int {
	switch grade {
	case KiloGrade:
		return 5000
	case FeatureGrade:
		return 15000
	default:
		return 40000
	}
}

// HeistPot: 一桌的獎池（全部入場費 × 1.5）。
func HeistPot(grade ShopGrade) int {
	return int(float64(HeistEntry(grade)*HeistSeats) * HeistPotMul)
}

// HeistPotBonusB: 保留給舊介面（稀有度已在入場費裡）。
func HeistPotBonusB(grade ShopGrade) float64 {
	return float64(HeistPot(grade)) / float64(HeistEntry(grade)*HeistSeats)
}

// HeistSeat: 一桌的一個位子（回合結算用）。
type HeistSeat struct {
	UserID int
	Name   string
	Alive  bool
	Action string
	Target int
	Entry  int
}

// HeistRound: 一輪的結算結果。
type HeistRound struct {
	Progress   int
	MutualPair int         // 互相合作的對數
	Deaths     map[int]int // 死者 → 兇手（0 = 崩塌/反殺）
	Looters    map[int]int // 兇手 → 取走的入場費
	Exposed    map[int]int // 受害者 → 想殺他的人（死裏逃生者知道是誰動的手）
	Collapse   bool        // 崩塌（全員背叛）
	Log        []string
}

// ResolveHeistRound: 純函式結算一輪。
func ResolveHeistRound(seats []HeistSeat, progress, target int, r Rand) HeistRound {
	_ = r // 投票式沒有隨機：勝負取決於讀心
	res := HeistRound{Progress: progress, Deaths: map[int]int{}, Looters: map[int]int{},
		Exposed: map[int]int{}}
	alive := []HeistSeat{}
	for _, s := range seats {
		if s.Alive {
			alive = append(alive, s)
		}
	}
	if len(alive) == 0 {
		return res
	}
	byID := map[int]HeistSeat{}
	for _, s := range alive {
		byID[s.UserID] = s
	}

	// 0) 崩塌：每個活人都投了背叛 → 全部死亡、獎池沒收
	betrayVotes := 0
	for _, s := range alive {
		if s.Action == HeistBetray {
			betrayVotes++
		}
	}
	if betrayVotes == len(alive) {
		res.Collapse = true
		for _, s := range alive {
			res.Deaths[s.UserID] = 0
		}
		res.Log = append(res.Log, fmt.Sprintf("全員同時開槍——礦坑崩塌，%d 人全部陪葬，獎池沒收", len(alive)))
		return res
	}

	// 1) 互相合作 → 進度（合作者兩兩成對）
	coop := 0
	for _, s := range alive {
		if s.Action == HeistCooperate {
			coop++
		}
	}
	res.MutualPair = coop * (coop - 1) / 2
	if res.MutualPair > 0 {
		res.Progress += res.MutualPair
		if res.Progress > target {
			res.Progress = target
		}
		res.Log = append(res.Log, fmt.Sprintf("%d 人互相合作，進度 +%d（%d/%d）", coop, res.MutualPair, res.Progress, target))
	}

	// 2) 背叛：沒被對方背叛的人死（同時結算）
	victims := map[int][]int{} // 死者 → 動手的人
	for _, s := range alive {
		if s.Action != HeistBetray || s.Target == 0 {
			continue
		}
		victim, ok := byID[s.Target]
		if !ok {
			continue
		}
		if victim.Action == HeistBetray && victim.Target == s.UserID {
			res.Log = append(res.Log, fmt.Sprintf("%s 與 %s 同時背叛，互相抵銷——兩個都沒死", s.Name, victim.Name))
			continue
		}
		victims[victim.UserID] = append(victims[victim.UserID], s.UserID)
	}
	for victimID, killers := range victims {
		v := byID[victimID]
		loot := int(float64(v.Entry) * HeistKillTake)
		share := loot / len(killers)
		for _, k := range killers {
			res.Looters[k] += share
			res.Exposed[victimID] = k // 死裏逃生（或身亡）的那位知道是誰動的手
		}
		res.Deaths[victimID] = killers[0]
		res.Log = append(res.Log, fmt.Sprintf("%s 被 %d 人圍殺，取走 %d 籌碼", v.Name, len(killers), loot))
	}
	return res
}

// HeistPayout: 結算獎池。alive = 倖存者；solo = 只剩一人獨吞。
func HeistPayout(grade ShopGrade, pot int, alive []int) map[int]int {
	out := map[int]int{}
	if len(alive) == 0 {
		return out
	}
	share := pot / len(alive)
	for _, uid := range alive {
		out[uid] = share
	}
	return out
}

// 入場費越高、難度越高（用戶指定）：每高一檔 +4 格進度、+1 回合上限。
func HeistTargetFor(grade ShopGrade) int { return HeistTargetBase + 4*int(grade) }
func HeistRoundsFor(grade ShopGrade) int { return HeistRounds + int(grade) }
