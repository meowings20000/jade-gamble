package domain

import "fmt"

// 稱號系統（2026-09-16）
//
// 稱號由「統計數字」解鎖（不另外存表，每次算），玩家可以裝備一個顯示在名字前面。
// 條件一律用玩家自己的紀錄算：切了幾刀、最好的倍率、圖鑑分數、籌碼、連續磚頭…
// 這樣新稱號只要加一條規則，舊玩家下次登入就自動有機會解鎖。

// TitleStats: 判斷稱號用的統計。
type TitleStats struct {
	Cuts        int     // 切石刀數
	Scratches   int     // 刮石次數
	Polishes    int     // 磨石落袋次數
	BestMult    float64 // 單刀最佳倍率
	WorstMult   float64 // 單刀最差倍率
	BrickStreak int     // 連續磚頭
	Chips       int     // 目前籌碼
	Collection  int     // 圖鑑分數
	Varieties   int     // 收集到的異色數
	Gems        int     // 切出的寶石彩蛋數
	Diamonds    int     // 切出的鑽石數
}

// Title: 一個稱號。
type Title struct {
	Key   string
	Name  string
	Desc  string
	Rare  int // 1=普通 2=少見 3=稀有 4=傳說（前端顯示顏色用）
	Check func(TitleStats) bool
}

// Titles: 由易到難。
var Titles = []Title{
	{"newbie", "見習賭徒", "開始賭石", 1, func(s TitleStats) bool { return true }},
	{"first_cut", "開山第一刀", "切開第一顆石頭", 1, func(s TitleStats) bool { return s.Cuts >= 1 }},
	{"ten_cuts", "十刀俠", "切開 10 顆石頭", 1, func(s TitleStats) bool { return s.Cuts >= 10 }},
	{"hundred_cuts", "千刀萬剮", "切開 100 顆石頭", 2, func(s TitleStats) bool { return s.Cuts >= 100 }},
	{"five_hundred", "石痴", "切開 500 顆石頭", 3, func(s TitleStats) bool { return s.Cuts >= 500 }},
	{"lucky", "幸運兒", "一刀切出 5 倍以上", 2, func(s TitleStats) bool { return s.BestMult >= 5 }},
	{"golden_eye", "黃金瞳", "當日連贏 5 刀，再切出 5 倍以上", 3, func(s TitleStats) bool { return s.BestMult >= 5 && s.Cuts >= 5 }},
	{"imperial", "帝王之眼", "親手切出帝王綠", 4, func(s TitleStats) bool { return s.Varieties >= 9 }},
	{"collector", "收藏家", "圖鑑分數 100 以上", 3, func(s TitleStats) bool { return s.Collection >= 100 }},
	{"lapidary", "琢磨師", "磨石落袋 20 次", 2, func(s TitleStats) bool { return s.Polishes >= 20 }},
	{"scratcher", "刮皮專家", "刮開 50 顆石頭", 2, func(s TitleStats) bool { return s.Scratches >= 50 }},
	{"gambler", "賭徒之魂", "連切 4 顆磚頭料", 2, func(s TitleStats) bool { return s.BrickStreak >= 4 }},
	{"gem_hunter", "寶石獵人", "切出非玉石的寶石", 3, func(s TitleStats) bool { return s.Gems >= 1 }},
	{"diamond", "鑽石之手", "切出鑽石", 4, func(s TitleStats) bool { return s.Diamonds >= 1 }},
	{"rich", "一方之霸", "籌碼 50 萬", 3, func(s TitleStats) bool { return s.Chips >= 500000 }},
	{"tycoon", "賭石大亨", "籌碼 200 萬", 4, func(s TitleStats) bool { return s.Chips >= 2000000 }},
	{"ruined", "傾家蕩產", "曾經輸到只剩幾百籌碼", 2, func(s TitleStats) bool { return s.Chips <= 200 }},
}

// TitleByKey: 依 key 取稱號。
func TitleByKey(key string) (Title, bool) {
	for _, t := range Titles {
		if t.Key == key {
			return t, true
		}
	}
	return Title{}, false
}

// UnlockedTitles: 這個統計解鎖了哪些稱號。
func UnlockedTitles(s TitleStats) []string {
	out := []string{}
	for _, t := range Titles {
		if t.Check(s) {
			out = append(out, t.Key)
		}
	}
	return out
}

// TitleView: 給前端的一列。
func TitleView(t Title, unlocked bool, equipped bool) map[string]any {
	return map[string]any{
		"key": t.Key, "name": t.Name, "desc": t.Desc, "rare": t.Rare,
		"unlocked": unlocked, "equipped": equipped,
	}
}

// TitleLine: 名字前面加上稱號（排行榜／聊天顯示用）。
func TitleLine(title, name string) string {
	if title == "" {
		return name
	}
	return fmt.Sprintf("【%s】%s", title, name)
}
