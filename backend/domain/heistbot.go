package domain

// 奪寶桌的 bot（2026-09-16）——桌上沒真人時補位，每隻脾氣不同。
//
// 傾向用三個參數描述：
//	CoopRate  這一輪「想合作」的機率（其餘想背叛）
//	GreedAt   進度到幾成時開始起貪念（進度 ≥ GreedAt 就大幅提高背叛機率）
//	Revenge   有人背叛過我，之後每輪優先殺他（記仇）

type HeistBot struct {
	Key      string
	Name     string
	Style    string
	CoopRate float64
	GreedAt  float64
	Revenge  bool
}

// HeistBots: 15 隻不同傾向的 bot（每桌隨機抽，不會一直看到同樣幾個）。
var HeistBots = []HeistBot{
	{"afu", "老實人阿福", "死忠合作派：幾乎不背叛，被背叛過一定報仇", 0.95, 0.99, true},
	{"xiaoli", "牆頭草小李", "前期乖乖合作，進度七成就開始想獨吞", 0.75, 0.70, false},
	{"daoke", "冷血刀客", "一有機會就動刀，只有三成會合作", 0.30, 0.20, false},
	{"ayi", "記仇的阿姨", "誰背叛過我，我這輩子都記得", 0.70, 0.85, true},
	{"tancai", "貪財叔", "只看獎池多大：進度越滿越想殺人", 0.60, 0.55, false},
	{"foxi", "佛系青年", "永遠合作，隨緣", 1.0, 1.01, false},
	{"shuangmian", "雙面人", "五五波，連自己都不知道下一步", 0.50, 0.60, false},
	{"baoan", "保全阿明", "看到有人被殺就翻臉", 0.80, 0.90, true},
	{"shuan", "算盤張", "賠率算到骨子裡：只有划算才背叛", 0.85, 0.75, false},
	{"jiu", "酒鬼老周", "喝多了亂投，四成機率背叛", 0.60, 0.45, false},
	{"xiaoqi", "小氣鬼琪琪", "進度沒到八成絕不冒險", 0.90, 0.80, true},
	{"mianrou", "麵店老闆", "誰不合作就跟他翻臉", 0.80, 0.65, true},
	{"laoshi", "退休老師", "講道理，但被騙過就報復", 0.88, 0.95, true},
	{"diandian", "點點", "新手，常常選錯目標", 0.65, 0.50, false},
	{"sanbai", "三百塊", "入場費越貴越想賭一把大的", 0.55, 0.35, false},
}

// PickHeistBots: 每桌隨機抽 n 隻（避免每桌都同一批人）。
func PickHeistBots(n int, r Rand) []HeistBot {
	pool := make([]HeistBot, len(HeistBots))
	copy(pool, HeistBots)
	for i := len(pool) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		pool[i], pool[j] = pool[j], pool[i]
	}
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

// HeistBotVote: 決定這隻 bot 這一輪怎麼投。
func HeistBotVote(b HeistBot, me HeistSeat, exposed int, others []HeistSeat, progress, target int, r Rand) (string, int) {
	if len(others) == 0 {
		return HeistCooperate, 0
	}
	// 1) 記仇：有人動過我 → 直接找他算帳（他還沒死的話）
	if b.Revenge && exposed > 0 {
		for _, o := range others {
			if o.UserID == exposed {
				return HeistBetray, o.UserID
			}
		}
	}
	// 2) 進度越滿越貪
	coopRate := b.CoopRate
	ratio := 0.0
	if target > 0 {
		ratio = float64(progress) / float64(target)
	}
	if ratio >= b.GreedAt {
		coopRate *= 0.35
	}
	if r.Float64() < coopRate {
		return HeistCooperate, 0
	}
	// 3) 挑目標：隨機挑一個活人下手
	return HeistBetray, others[r.Intn(len(others))].UserID
}
