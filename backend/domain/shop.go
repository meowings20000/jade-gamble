package domain

// Shop pricing: personal shelves, free daily restock, escalating refresh.

type ShelfSpec struct {
	Grade     ShopGrade
	Slots     int
	BasePrice int // first refresh of the day
}

var shelfSpecs = map[ShopGrade]ShelfSpec{
	KiloGrade:    {KiloGrade, 8, 300},
	FeatureGrade: {FeatureGrade, 6, 1000},
	WindowGrade:  {WindowGrade, 4, 5000},
}

// ShelfSize returns the slot count per grade.
func ShelfSize(g ShopGrade) int { return shelfSpecs[g].Slots }

// RefreshPrice: cost of the (n+1)-th refresh today for a grade.
// Doubles each time: base, ×2, ×4, ×8… capped at ×16.
func RefreshPrice(g ShopGrade, refreshesToday int) int {
	base := shelfSpecs[g].BasePrice
	mult := 1
	for i := 0; i < refreshesToday && i < 4; i++ { // cap at 16×
		mult *= 2
	}
	return base * mult
}

// CutReveal: 切石 one-shot result. Double coupon doubles payout (x0 stays 0
// via Brick check upstream). Returns payout and full reveal payload.
func CutReveal(s *Stone, doubleCoupon bool) (payout int) {
	payout = s.BaseValue()
	if doubleCoupon && s.Quality != Brick && payout > 0 {
		payout *= 2
	}
	return payout
}

// CutRevealWithGem: 正式切石流程——先擲隱藏彩蛋（5% 寶石），沒中就正常結算。
// 舊的 CutReveal 保持「不含彩蛋」的純計算，讓既有測試與定價邏輯穩定。
func CutRevealWithGem(s *Stone, doubleCoupon bool, r Rand) (payout int, gem *Gem) {
	if g, ok := RollGem(r); ok {
		return GemPayout(s, g, doubleCoupon), &g
	}
	return CutReveal(s, doubleCoupon), nil
}

// Redemption catalog (兌換所). EV of every consumable is below its price.
type ExchangeItem struct {
	Key         string
	Name        string
	Price       int
	Kind        string // "consumable" | "buff" | "cosmetic"
	Description string
}

var ExchangeCatalog = []ExchangeItem{
	{"frenzy_ticket", "刮到爽門票", 2000, "consumable",
		"10 顆休閒石立即開刮，每顆掉 1~100 喵喵幣，純解壓。"},
	{"insurance", "保險券", 0, "consumable",
		"磨崩退回 50% 石底價。價格＝磨石標的底價的 15%（購買時自動計）。"},
	{"double_coupon", "雙倍券", 3000, "consumable",
		"下一刀切石賠率 ×2（磚頭料不救）。"},
	{"free_refresh", "免費刷新券", 5000, "consumable",
		"任一檔貨架免費刷新一次（當日遞增次數照算，適合價格已經漲上去時用）。"},
	{"light_master", "打燈大師卡", 2000, "buff",
		"24 小時打燈報告半價，誤導率 30%→22%。"},
	{"golden_eye", "黃金瞳殘光", 10000, "buff",
		"1 小時表現料皮殼多標註一條真實特徵。"},
	{"polish_touch", "磨石手感", 5000, "buff",
		"1 小時磨崩機率 −8%。"},
	{"frame_gold", "賭場金頭像框", 2500, "cosmetic", "純裝飾：金邊頭像。"},
	{"frame_imperial", "帝王綠頭像框", 5000, "cosmetic", "純裝飾：帝王綠光暈。"},
	{"frame_cat", "喵喵框", 1200, "cosmetic", "純裝飾：貓耳粉邊（最便宜的一框）。"},
	{"frame_violet", "紫羅蘭框", 3500, "cosmetic", "純裝飾：紫羅蘭色光暈。"},
	{"frame_ink", "墨翠框", 6000, "cosmetic", "純裝飾：墨翠沉綠。"},
	{"frame_rainbow", "虹彩框", 12000, "cosmetic", "純裝飾：七彩流動，收藏級。"},
	{"theme_jade", "賭桌配色・翠玉", 3000, "cosmetic", "把整個介面換成翠玉桌布（深綠玉色）。"},
	{"theme_violet", "賭桌配色・紫氣", 5000, "cosmetic", "紫羅蘭夜色桌布。"},
	{"theme_ink", "賭桌配色・墨玉", 7000, "cosmetic", "墨翠沉綠，最耐看。"},
	{"theme_gold", "賭桌配色・金碧", 10000, "cosmetic", "金碧輝煌，土豪專用。"},
	{"fx_confetti", "開箱彩帶特效", 4000, "cosmetic", "切石／落袋賺大錢時噴彩帶（靠自己打出好料才會噴）。"},
}

func ExchangeItemByKey(key string) (ExchangeItem, bool) {
	for _, it := range ExchangeCatalog {
		if it.Key == key {
			return it, true
		}
	}
	return ExchangeItem{}, false
}

// FrenzyTicketPayout: one casual stone pays 1..100, EV 60.
func FrenzyTicketPayout(r Rand) int { return 1 + r.Intn(100) }

// BankruptcyRelief options.
// SignupChips: 開局籌碼（2026-09-16 由 1 萬提高到 5 萬）。
const SignupChips = 50000

const (
	// ReliefThreshold: 籌碼低於此數即可領救濟。
	ReliefThreshold = 5000
	// ReliefPerDay: 每 24 小時最多領幾次救濟（2026-09-17 用戶要求：1 天 3 次）。
	ReliefPerDay   = 3
	ReliefChips    = 50000
	ReliefAltChips = 300
)

// DailyAllowance: chips granted on a brand-new day (loyalty drip).
const DailyAllowance = 500
