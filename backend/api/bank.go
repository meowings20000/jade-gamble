package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// 喵喵錢莊 API（2026-09-16）
//
// AI（DeepSeek，key 由後端 .env 提供，永不進 repo）扮演貓娘老闆娘審核借款與申訴；
// 沒有 key 或 AI 出錯時，退回 domain.FallbackReview 的規則式審核，遊戲不會卡住。

// bank: GET /api/bank —— 我的貸款狀態、對話、歷史與條款。
func (a *API) bank(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	a.seizeOverdue() // 進錢莊頁面時順手處理逾期
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	loan, err := a.Store.ActiveLoan(uid)
	if err != nil {
		return err
	}
	offer, _ := a.Store.PendingOffer(uid)
	chat := []map[string]string{}
	var repayTotal int
	if loan != nil {
		chat, _ = a.Store.LoanChat(loan.ID)
		repayTotal = loan.Principal + loan.Interest
	} else if offer != nil {
		// 還沒按「接受」時，對話記在「提議」那筆上（申訴來回都在這裡）
		chat, _ = a.Store.LoanChat(offer.ID)
	}
	hist, err := a.Store.LoanHistory(uid, 10)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"chips": u.Chips, "loan": loan, "offer": offer, "chat": chat, "history": hist,
		"repay_total": repayTotal,
		"terms": map[string]any{
			"min": domain.BankMinAmount, "max": domain.BankMaxAmount,
			"min_hours": domain.BankMinHours, "max_hours": domain.BankMaxHours,
			"rate": domain.BankRate, "max_rate": domain.BankMaxRate,
			"seize_ratio": domain.BankSeizeRatio, "max_appeal": domain.BankMaxAppeal,
			"ai": a.AIEnabled(),
		},
		"greeting": "歡迎光臨喵喵錢莊喵～要借多少、甚麼時候還，自己說清楚喵。",
	})
	return nil
}

// bankApply: POST /api/bank/apply {amount, hours, reason}
func (a *API) bankApply(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Amount int     `json:"amount"`
		Hours  int     `json:"hours"`
		Rate   float64 `json:"rate"` // 玩家自己提的利率（可留空＝25%）
		Reason string  `json:"reason"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	if cur, _ := a.Store.ActiveLoan(uid); cur != nil {
		return errors.New("你上一筆還沒還清喵")
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if msg := domain.ValidateLoanRequest(body.Amount, body.Hours, u.Chips); msg != "" {
		return errors.New(msg)
	}

	if body.Rate < domain.BankRate {
		body.Rate = domain.BankRate
	}
	if body.Rate > domain.BankMaxRate {
		body.Rate = domain.BankMaxRate
	}
	dec, err := a.catReview(u.Username, u.Chips, body.Amount, body.Hours, body.Rate, body.Reason, 0, nil)
	if err != nil {
		return err
	}
	if dec.Decision == "deny" {
		// 被拒絕 → 記錄下來，玩家可以申訴（最多 3 輪）
		if err := a.Store.WithTx(func(tx *store.Tx) error {
			id, err := a.Store.SaveApplicationTx(tx, uid, body.Amount, body.Hours, body.Rate, body.Reason)
			if err != nil {
				return err
			}
			return a.Store.LoanChatTx(tx, id, "banker", dec.Message)
		}); err != nil {
			return err
		}
	}
	return a.proposeOffer(w, uid, body.Amount, body.Hours, body.Rate, body.Reason, dec, 0)
}

// bankAppeal: POST /api/bank/appeal {message} —— 最多 3 輪。
func (a *API) bankAppeal(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		return errors.New("要說點甚麼喵")
	}
	u, err := a.Store.GetUser(uid)
	if err != nil {
		return err
	}
	// 申訴對象：等玩家答覆的「提議」或「被拒絕的申請」——滿意的話直接接受就好，
	// 不滿意（覺得利率太高／金額太少）才來申訴，讓貓娘心軟慢慢降。
	loan, err := a.Store.PendingOffer(uid)
	if err != nil {
		return err
	}
	if loan == nil {
		loan, _ = a.Store.DeniedApplication(uid)
	}
	if loan == nil {
		return errors.New("現在沒有可以申訴的提議喵")
	}
	if loan.Appeals >= domain.BankMaxAppeal {
		return errors.New("申訴次數用完了喵（最多 3 輪）")
	}
	hist, _ := a.Store.LoanChat(loan.ID)
	history := []string{}
	for _, m := range hist {
		who := "玩家"
		if m["role"] == "banker" {
			who = "錢莊"
		}
		history = append(history, who+"："+m["content"])
	}
	history = append(history, "玩家："+body.Message)

	dec, err := a.catReview(u.Username, u.Chips, loan.Principal, loan.Hours, loan.Rate, loan.Reason, loan.Appeals+1, history)
	if err != nil {
		return err
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if err := a.Store.BumpAppealTx(tx, loan.ID); err != nil {
			return err
		}
		if err := a.Store.LoanChatTx(tx, loan.ID, "player", body.Message); err != nil {
			return err
		}
		return a.Store.LoanChatTx(tx, loan.ID, "banker", dec.Message)
	}); err != nil {
		return err
	}
	granted, bal, repayTotal, err := a.decideLoanAction(uid, loan, dec)
	if err != nil {
		return err
	}
	updated, _ := a.Store.DeniedApplication(uid)
	if updated == nil {
		updated, _ = a.Store.ActiveLoan(uid)
	}
	out := map[string]any{"ok": true, "decision": dec.Decision, "message": dec.Message,
		"loan": updated, "appeals_left": domain.BankMaxAppeal - (loan.Appeals + 1), "granted": granted}
	if granted {
		out["chips"] = bal
		out["repay_total"] = repayTotal
	}
	writeJSON(w, 200, out)
	return nil
}

// bankRepay: POST /api/bank/repay —— 還清本息。
func (a *API) bankRepay(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	loan, err := a.Store.ActiveLoan(uid)
	if err != nil || loan == nil {
		return errors.New("你沒有要還的貸款喵")
	}
	total := loan.Principal + loan.Interest + loan.Penalty
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, -total)
		if err != nil {
			return err
		}
		bal = b
		return a.Store.SetLoanStatusTx(tx, loan.ID, "repaid")
	}); err != nil {
		return errors.New(fmt.Sprintf("籌碼不夠還喵（要 %d）", total))
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal,
		"message": fmt.Sprintf("還清 %d 喵，算你守信用喵。", total)})
	return nil
}

// ---------- 內部 ----------

// proposeOffer: AI 的判斷一律先變成「提議」，玩家自己按接受或拒絕才放款。
func (a *API) proposeOffer(w http.ResponseWriter, uid, amount, hours int, rate float64, reason string, dec domain.BankDecision, round int) error {
	if dec.Decision == "deny" {
		return a.denyResponse(w, dec)
	}
	amt, hrs := amount, hours
	if dec.Decision == "counter" && dec.Amount > 0 {
		amt = dec.Amount
	}
	if dec.Hours > 0 {
		hrs = dec.Hours
	}
	r := domain.BankRateFor(dec.Rate)
	if dec.Decision == "waive" {
		r = r / 2 // 心軟：利率砍半
	}
	interest := domain.BankInterest(amt, r)
	penalty := dec.Penalty
	var offerID int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		id, err := a.Store.OfferLoanTx(tx, uid, amt, interest, hrs, r, penalty, reason)
		if err != nil {
			return err
		}
		offerID = id
		return a.Store.LoanChatTx(tx, offerID, "banker", dec.Message)
	}); err != nil {
		return err
	}
	offer, _ := a.Store.LoanByID(offerID)
	writeJSON(w, 200, map[string]any{"ok": true, "decision": dec.Decision, "message": dec.Message,
		"offer": offer, "need_accept": true, "repay_total": amt + interest})
	return nil
}

// bankAccept: POST /api/bank/accept —— 接受提議（這一步才入賬）。
func (a *API) bankAccept(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	offer, err := a.Store.PendingOffer(uid)
	if err != nil || offer == nil {
		return errors.New("現在沒有等你的提議喵")
	}
	if cur, _ := a.Store.ActiveLoan(uid); cur != nil {
		return errors.New("你上一筆還沒還清喵")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, offer.Principal)
		if err != nil {
			return err
		}
		bal = b
		if err := a.Store.LoanChatTx(tx, offer.ID, "player", "好，我接受喵。"); err != nil {
			return err
		}
		return a.Store.AcceptOfferTx(tx, offer.ID, offer.Hours)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal,
		"message": fmt.Sprintf("成交！%d 籌碼已入袋喵，到期還 %d。", offer.Principal, offer.Principal+offer.Interest)})
	return nil
}

// bankReject: POST /api/bank/reject —— 拒絕提議。
func (a *API) bankReject(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	offer, err := a.Store.PendingOffer(uid)
	if err != nil || offer == nil {
		return errors.New("現在沒有等你的提議喵")
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if err := a.Store.LoanChatTx(tx, offer.ID, "player", "算了，我不借了喵。"); err != nil {
			return err
		}
		return a.Store.SetLoanStatusTx(tx, offer.ID, "rejected")
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "好啦，不借就不借喵（甩尾）"})
	return nil
}

func (a *API) denyResponse(w http.ResponseWriter, dec domain.BankDecision) error {
	writeJSON(w, 200, map[string]any{"ok": false, "decision": "deny", "message": dec.Message})
	return nil
}

// decideLoanAction: 申訴結果——通過就把「被拒的申請」轉成新提議，等玩家按接受。
func (a *API) decideLoanAction(uid int, loan *domain.Loan, dec domain.BankDecision) (bool, int, int, error) {
	if dec.Decision == "deny" {
		return false, 0, 0, nil
	}
	amt, hrs := loan.Principal, loan.Hours
	if dec.Decision == "counter" && dec.Amount > 0 {
		amt = dec.Amount
	}
	if dec.Hours > 0 {
		hrs = dec.Hours
	}
	// 申訴：可以低於 30% 的開頭底線（心軟慢慢降），最低 10%
	rate := domain.BankRateForNegotiated(dec.Rate)
	if dec.Rate <= 0 {
		// 模型只把利率寫在話裡、JSON 沒帶 rate → 沿用上一輪再降一階（她一向是這樣說的）
		rate = loan.Rate - domain.BankAppealCut
		if rate < domain.BankMinNegotiated {
			rate = domain.BankMinNegotiated
		}
	}
	if rate > loan.Rate && dec.Penalty == 0 {
		rate = loan.Rate // 沒有理由就不會加價
	}
	interest := domain.BankInterest(amt, rate)
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		newID, err := a.Store.OfferLoanTx(tx, uid, amt, interest, hrs, rate, dec.Penalty, loan.Reason)
		if err != nil {
			return err
		}
		// 申訴次數帶著走：新提議繼承「已用輪數 + 1」，不然每輪都重置＝無限申訴
		if err := a.Store.SetLoanAppealsTx(tx, newID, loan.Appeals+1); err != nil {
			return err
		}
		return a.Store.SetLoanStatusTx(tx, loan.ID, "appealed")
	}); err != nil {
		return false, 0, 0, err
	}
	return true, 0, amt + interest, nil
}

// AIEnabled: 後端有沒有拿到 AI 金鑰（前端不碰 key）。
func (a *API) AIEnabled() bool { return a.AIAPIKey != "" && a.AIBaseURL != "" }

// catReview: 請貓娘老闆娘裁決；沒有 key 或失敗就用規則式審核。
func (a *API) catReview(user string, chips, amount, hours int, offeredRate float64, reason string, round int, history []string) (domain.BankDecision, error) {
	if !a.AIEnabled() {
		return domain.FallbackReview(user, chips, amount, hours, offeredRate, reason, round), nil
	}
	prompt := domain.BankReviewPrompt(user, chips, amount, hours, reason, round, history)
	payload := map[string]any{
		"model": a.AIModel,
		"messages": []map[string]string{
			{"role": "system", "content": domain.BankPersona()},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.8,
		"max_tokens":  400,
	}
	buf, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", strings.TrimRight(a.AIBaseURL, "/")+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return domain.FallbackReview(user, chips, amount, hours, offeredRate, reason, round), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.AIAPIKey)
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return domain.FallbackReview(user, chips, amount, hours, offeredRate, reason, round), nil
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Choices) == 0 {
		return domain.FallbackReview(user, chips, amount, hours, offeredRate, reason, round), nil
	}
	dec := parseBankDecision(out.Choices[0].Message.Content)
	return domain.NormalizeDecision(dec, user, chips, amount, hours, reason, round), nil
}

// parseBankDecision: 從模型輸出裡挖出 JSON（模型常常包 ```json ```）。
func parseBankDecision(s string) domain.BankDecision {
	var d domain.BankDecision
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return domain.BankDecision{}
	}
	return d
}

// seizeOverdue: 逾期的貸款沒收一半財產（籌碼一半 + 倉庫一半石頭，挑最便宜的）。
func (a *API) seizeOverdue() {
	loans, err := a.Store.OverdueLoans()
	if err != nil {
		return
	}
	for _, l := range loans {
		// 注意：MaxOpenConns(1)，tx 裡不能再呼叫非 tx 的 store 方法（會死鎖）。
		// 所以先把要用的資料讀出來，再進交易。
		u, err := a.Store.GetUser(l.UserID)
		if err != nil {
			continue
		}
		ids, err := a.Store.OwnedStoneIDs(l.UserID)
		if err != nil {
			ids = nil
		}
		seize := u.Chips / 2
		take := len(ids) / 2
		_ = a.Store.WithTx(func(tx *store.Tx) error {
			if seize > 0 {
				if _, err := store.UpdateChipsTx(tx, l.UserID, -seize); err != nil {
					_, _ = store.UpdateChipsTx(tx, l.UserID, -u.Chips)
				}
			}
			for i := 0; i < take; i++ {
				_ = a.Store.SetStoneStateTx(tx, ids[i], domain.StateUsed, 0)
			}
			return a.Store.SetLoanStatusTx(tx, l.ID, "defaulted")
		})
	}
}
