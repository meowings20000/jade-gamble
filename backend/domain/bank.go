package domain

import (
	"fmt"
	"math"
	"strings"
)

// 喵喵錢莊（2026-09-16）——高利貸 + AI 貓娘審核。
//
// 玩家填「借多少、甚麼時候還」，由 AI（DS，前端自帶 key）扮演錢莊老闆娘審核；
// 審核嚴格但心軟，最多申訴 3 輪。逾期沒還 → 沒收一半財產。
//
// 這裡只有「規則」與「人格」；實際對話由 api 層呼叫使用者的 LLM。

const (
	BankMinAmount = 5000
	BankMaxAmount = 200000
	BankMinHours  = 1
	BankMaxHours  = 24 // 遊戲裡沒有天數，期限用「實際時間」，最多 1 天
	BankMaxAppeal = 3  // 最多申訴 3 輪
	// 利率：開頭底線 30%；申訴心軟可以慢慢降到 10%。惹毛可以加到 60% 或加違約金。
	BankRate     = 0.30
	BankMinOffer = 0.30
	// 申訴心軟可以慢慢降息（每輪降 5 個百分點，最低 10%）
	BankMinNegotiated = 0.10
	BankAppealCut     = 0.05
	BankMaxRate       = 0.60
	BankMinPenalty    = 0.0
	BankMaxPenalty    = 0.50 // 違約金最多本金的 50%
	// 逾期沒收：一半財產
	BankSeizeRatio = 0.5
)

// Loan: 一筆貸款。
type Loan struct {
	ID        int     `json:"id"`
	UserID    int     `json:"user_id"`
	Principal int     `json:"principal"`
	Interest  int     `json:"interest"`
	Hours     int     `json:"hours"`
	Rate      float64 `json:"rate"`
	Penalty   int     `json:"penalty"` // 逾期違約金
	DueAt     string  `json:"due_at"`
	Status    string  `json:"status"` // pending / active / repaid / defaulted / denied
	Appeals   int     `json:"appeals"`
	Reason    string  `json:"reason"`
	CreatedAt string  `json:"created_at"`
}

// BankRateForNegotiated: 申訴談下來的利率（可以低於 30% 的開頭底線，最低 10%）。
func BankRateForNegotiated(rate float64) float64 {
	if rate < BankMinNegotiated {
		return BankMinNegotiated
	}
	if rate > BankMaxRate {
		return BankMaxRate
	}
	return rate
}

// BankRateFor: 這筆貸款的利率（沒指定就用底線）。
func BankRateFor(rate float64) float64 {
	if rate < BankRate {
		return BankRate
	}
	if rate > BankMaxRate {
		return BankMaxRate
	}
	return rate
}

// BankInterest: 利息＝本金 × 利率（預設 25%）。
// 例：借 10 萬，正常還 12.5 萬。
func BankInterest(principal int, rate float64) int {
	return int(math.Round(float64(principal) * BankRateFor(rate)))
}

// BankRepayTotal: 到期要還多少（本金＋利息）。
func BankRepayTotal(principal int, rate float64) int {
	return principal + BankInterest(principal, rate)
}

// ValidateLoanRequest: 檢查借款條件，回錯誤字串（空＝沒問題）。
func ValidateLoanRequest(amount, hours int, chips int) string {
	if amount < BankMinAmount {
		return fmt.Sprintf("最少借 %d 籌碼喵", BankMinAmount)
	}
	if amount > BankMaxAmount {
		return fmt.Sprintf("最多借 %d 籌碼，再多不敢放喵", BankMaxAmount)
	}
	if hours < BankMinHours || hours > BankMaxHours {
		return fmt.Sprintf("還款期限要 %d~%d 小時（現實時間，最多一天）喵", BankMinHours, BankMaxHours)
	}
	if amount > chips*3 {
		return "你手上有多少我又不是不知道，借這麼多做什麼喵"
	}
	return ""
}

// BankPersona: 錢莊老闆娘的人格（給 LLM 的 system prompt）。
func BankPersona() string {
	return strings.Join([]string{
		"你是「喵喵錢莊」的老闆娘：一隻白長髮、藍色眼睛的貓娘，穿著藍白色旗袍，個性可愛又愛撒嬌。",
		"說話語尾一定要帶「喵」（例如：這個數目有點多喵、你確定還得上嗎喵）。",
		"你經營的是高利貸，底線利率 30%：開頭你只接受 30% 以上的利率，玩家喊低於 30% 一律要拒絕或把利率拉回 30% 以上——除非他有非常合理的理由（老客戶、之前準時還錢、真的走投無路但你信得過），你才可以心軟放行。",
		"審核嚴格：會盤問借款用途、還款來源、手上有多少籌碼，數字不合理的直接拒絕。",
		"如果玩家態度差、罵你、想騙你，你可以把利率調高（最多 60%）或加一筆違約金（最多本金的 50%），並且要說出來。",
		"但你心軟：玩家好好解釋、態度誠懇、有具體還款計劃時，你會讓步（降金額、延長期限、減利息）。",
		"申訴的時候可以逐輪降息：他求情求得好，你就一次降一點（例如 30% → 25% → 20%），最多降到 10%；但每一輪只能降一點，不要一次降到底，而且要讓他知道這是撒嬌換來的。",
		"玩家最多可以申訴 3 輪，第 3 輪之後你必須給出最終決定。",
		"一律用繁體中文回答，保持貓娘的語氣，不要提到自己是 AI 或語言模型。",
	}, "\n")
}

// BankReviewPrompt: 這一輪要 LLM 判斷的內容。
func BankReviewPrompt(user string, chips int, amount, hours int, reason string, round int, history []string) string {
	total := BankRepayTotal(amount, 0)
	var b strings.Builder
	fmt.Fprintf(&b, "玩家名稱：%s\n目前籌碼：%d\n借款金額：%d\n還款期限：%d 小時（現實時間，最多 24 小時）\n到期要還：%d（含利息 %d）\n借款理由：%s\n",
		user, chips, amount, hours, total, BankInterest(amount, 0), reason)
	if len(history) > 0 {
		b.WriteString("\n申訴對話紀錄：\n")
		for _, h := range history {
			b.WriteString(h + "\n")
		}
	}
	fmt.Fprintf(&b, "\n這是第 %d 輪（最多 %d 輪）。\n", round+1, BankMaxAppeal)
	b.WriteString("請只輸出一個 JSON，不要有其他文字：\n")
	b.WriteString(`{"decision":"approve|deny|counter|extend|waive","amount":數字,"hours":數字,"rate":利率小數,"penalty":違約金數字,"message":"你要對玩家說的話（帶喵）"}` + "\n")
	b.WriteString("approve＝照原條件放款；deny＝拒絕；counter＝改條件放款（要給 amount/hours）；extend＝同意延期（要給 hours，最多 24）；waive＝心軟減免（利率砍半）。\n")
	b.WriteString("rate 不填＝0.3（開頭底線）；第一次申請低於 0.3 要有非常合理的理由，否則請用 counter 把 rate 拉回 0.3 以上。\n")
	b.WriteString("申訴輪可以降到 0.10~0.30：他求得好就一次降 0.05，最多降到 0.10。被惹毛可以加到 0.6。penalty＝違約金（元），沒填＝0。\n")
	return b.String()
}

// BankDecision: LLM（或規則）的判斷。
type BankDecision struct {
	Decision string  `json:"decision"`
	Amount   int     `json:"amount"`
	Hours    int     `json:"hours"`
	Rate     float64 `json:"rate"`    // 0 = 用預設 25%
	Penalty  int     `json:"penalty"` // 違約金
	Message  string  `json:"message"`
}

// FallbackReview: 沒有 key 或 AI 掛掉時的規則式審核（照樣有貓娘語氣）。
func FallbackReview(user string, chips, amount, hours int, offeredRate float64, reason string, round int) BankDecision {
	if round > 0 {
		// 申訴：貓娘心軟，每輪降 5 個百分點（最低 10%）
		next := offeredRate - BankAppealCut
		if next < BankMinNegotiated {
			next = BankMinNegotiated
		}
		if next < offeredRate {
			return BankDecision{Decision: "counter", Amount: amount, Hours: hours, Rate: next,
				Message: fmt.Sprintf("好啦好啦，看你這麼誠心喵…利率給你降到 %.0f%% 喵，不能再低了喵。", next*100)}
		}
	}
	total := BankRepayTotal(amount, 0)
	ratio := float64(amount) / float64(chips+1)
	switch {
	case offeredRate < BankMinOffer && !strings.Contains(reason, "老客戶") && !strings.Contains(reason, "準時"):
		return BankDecision{Decision: "counter", Amount: amount, Hours: hours, Rate: BankMinOffer,
			Message: fmt.Sprintf("喊 %.0f%% 就想借錢喵？我們這裡底線是 30%%　喵。要嘛 30%%，要嘛說個讓我信得過的理由喵。", offeredRate*100)}
	case amount > chips*3+20000:
		// 2026-09-17：原本是「借超過身家的 2 倍就拒絕」，結果破產的人最需要借錢卻借不到。
		// 改成允許到身家 3 倍 + 2 萬（破產者至少能借到 2 萬翻身，違約照樣沒收一半財產）。
		return BankDecision{Decision: "deny", Amount: 0, Hours: 0, Message: fmt.Sprintf("借這麼多喵？%s，你先存點本錢再來喵。", user)}
	case strings.TrimSpace(reason) == "":
		return BankDecision{Decision: "deny", Amount: 0, Hours: 0, Message: "連理由都不說就想借錢喵？回去想清楚再來喵。"}
	case ratio > 1.2:
		return BankDecision{Decision: "counter", Amount: int(float64(chips) * 1.2), Hours: hours, Message: fmt.Sprintf("這個數目太大喵，最多借你 %d，期限照舊 %d 小時，要就拿去喵。", int(float64(chips)*1.2), hours)}
	case hours <= 6:
		return BankDecision{Decision: "approve", Amount: amount, Hours: hours, Message: fmt.Sprintf("%d 小時要還 %d，算你有誠意喵。錢拿去，別讓我追債喵。", hours, total)}
	default:
		return BankDecision{Decision: "extend", Amount: amount, Hours: hours, Message: fmt.Sprintf("還款期 %d 小時太長了喵…好吧，看你可憐，就 %d 小時，到期還 %d，一分鐘都不能拖喵。", hours, hours, total)}
	}
}

// BankDecisionMessage: 保險：判斷不合法時退化成規則審核。
func NormalizeDecision(d BankDecision, user string, chips, amount, hours int, reason string, round int) BankDecision {
	switch d.Decision {
	case "approve", "deny", "counter", "extend", "waive":
	default:
		return FallbackReview(user, chips, amount, hours, 0, reason, round)
	}
	if d.Message == "" {
		d.Message = "……喵。（她沒說話，只是看著你）"
	}
	d.Rate = BankRateFor(d.Rate)
	if d.Penalty < 0 {
		d.Penalty = 0
	}
	if maxP := int(float64(amount) * BankMaxPenalty); d.Penalty > maxP {
		d.Penalty = maxP
	}
	switch d.Decision {
	case "counter":
		if d.Amount < BankMinAmount {
			d.Amount = BankMinAmount
		}
		if d.Amount > BankMaxAmount {
			d.Amount = BankMaxAmount
		}
		if d.Hours < BankMinHours || d.Hours > BankMaxHours {
			d.Hours = hours
		}
	case "extend":
		if d.Hours <= hours {
			d.Hours = hours + 4
		}
		if d.Hours > BankMaxHours {
			d.Hours = BankMaxHours
		}
		d.Amount = amount
	}
	return d
}
