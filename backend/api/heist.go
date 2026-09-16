package api

import (
	"errors"
	"fmt"
	"math"
	"net/http"

	"jade-gamble/backend/domain"
	"jade-gamble/backend/store"
)

// 奪寶 API（2026-09-16）——四人囚徒困境。
//
// GET  /api/heist        我的桌子狀態（含進度、對手、誰想殺我）
// POST /api/heist/join   {grade} 入場（扣入場費，湊滿 4 人開局）
// POST /api/heist/act    {action, target} 這一輪：合作 / 警戒 / 背叛
// POST /api/heist/leave  退出（還沒開局可以退，開局後算放棄）

// heistState: GET /api/heist
func (a *API) heistState(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	h, err := a.Store.MyHeist(uid)
	if err != nil {
		return err
	}
	if h != nil && h.Status == "running" {
		// 進到大廳/桌子就把逾時的回合推走（不然有人掛機全桌卡死）
		if err := a.heistTimeout(h); err != nil {
			_ = err
		}
		if h2, err := a.Store.HeistByID(h.ID); err == nil && h2 != nil {
			h = h2
		}
	}
	if h == nil {
		// 剛結束的桌子也回傳，讓玩家看到結算（不然畫面會直接彈回大廳）
		if rec, err := a.Store.MyHeistRecent(uid, 30); err == nil && rec != nil && rec.Status == "done" {
			h = rec
		}
	}
	if h != nil {
		// 等 20 秒沒人來 → 補 bot 開局（每桌隨機抽 15 隻裡的一部分）
		if err := a.maybeFillBots(h); err != nil {
			writeJSON(w, 200, map[string]any{"state_error": "fill: " + err.Error()})
			return nil
		}
		if h2, err := a.Store.HeistByID(h.ID); err == nil && h2 != nil {
			h = h2
		}
	}
	out := map[string]any{
		"fees": map[string]int{
			"kilo": domain.HeistEntry(domain.KiloGrade), "feature": domain.HeistEntry(domain.FeatureGrade),
			"window": domain.HeistEntry(domain.WindowGrade),
		},
		"pots": map[string]int{
			"kilo": domain.HeistPot(domain.KiloGrade), "feature": domain.HeistPot(domain.FeatureGrade),
			"window": domain.HeistPot(domain.WindowGrade),
		},
		"kill_take": domain.HeistKillTake, "pot_mul": domain.HeistPotMul,
		"rounds": domain.HeistRoundsFor(domain.WindowGrade), "target": domain.HeistTargetFor(domain.WindowGrade),
		"tiers": []map[string]any{
			{"grade": int(domain.KiloGrade), "name": "公斤料", "fee": domain.HeistEntry(domain.KiloGrade),
				"pot": domain.HeistPot(domain.KiloGrade), "target": domain.HeistTargetFor(domain.KiloGrade), "rounds": domain.HeistRoundsFor(domain.KiloGrade)},
			{"grade": int(domain.FeatureGrade), "name": "表現料", "fee": domain.HeistEntry(domain.FeatureGrade),
				"pot": domain.HeistPot(domain.FeatureGrade), "target": domain.HeistTargetFor(domain.FeatureGrade), "rounds": domain.HeistRoundsFor(domain.FeatureGrade)},
			{"grade": int(domain.WindowGrade), "name": "開窗料", "fee": domain.HeistEntry(domain.WindowGrade),
				"pot": domain.HeistPot(domain.WindowGrade), "target": domain.HeistTargetFor(domain.WindowGrade), "rounds": domain.HeistRoundsFor(domain.WindowGrade)},
		},
		"rules": "四人一桌、30 秒一輪（沒出手＝自動合作）。互相合作才推進度（四人全合作 +6 格/輪）。" +
			"入場費越高越難：公斤料 18 格／5 輪、表現料 22 格／6 輪、開窗料 26 格／7 輪。" +
			"每人一輪一張背叛票：對方沒背叛你 → 他死、你拿他 70% 入場費；對方也背叛你 → 互相抵銷，兩個都沒死（但他知道你想殺他）。" +
			"全員同時背叛＝礦坑崩塌，全部陪葬、獎池沒收；只剩一人獨吞 6 倍入場費。",
	}
	// 大廳：誰在排隊、還缺幾人成團
	if qs, err := a.Store.OpenHeistQueues(); err == nil {
		list := []map[string]any{}
		for _, q := range qs {
			if len(q.Names) == 0 {
				continue // 空桌不顯示（沒人在排）
			}
			missing := domain.HeistSeats - len(q.Names)
			if missing < 0 {
				missing = 0
			}
			entry := map[string]any{"id": q.ID, "grade": q.Grade, "entry": q.Entry, "status": q.Status,
				"names": q.Names, "count": len(q.Names), "missing": missing, "need": domain.HeistSeats,
				"wait_sec": int(q.AgeSec), "bot_in": domain.HeistBotWaitSec - int(q.AgeSec)}
			if entry["bot_in"].(int) < 0 {
				entry["bot_in"] = 0
			}
			list = append(list, entry)
		}
		out["queues"] = list
	}
	if h == nil {
		writeJSON(w, 200, out)
		return nil
	}
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		writeJSON(w, 200, map[string]any{"state_error": "seats: " + err.Error()})
		return nil
	}
	mine, others := map[string]any{}, []map[string]any{}
	for _, s := range seats {
		item := map[string]any{
			"user_id": s.UserID, "name": s.Name, "alive": s.Alive, "acted": s.Action != "",
			"entry": s.Entry, "payout": s.Payout, "killed_by": s.KilledBy,
			// 誰想殺我：只有「死裏逃生」的那位看得到兇手身份
			"tried_to_kill_me": s.Exposed,
		}
		if s.UserID == uid {
			mine = item
			mine["my_action"] = s.Action
		} else {
			others = append(others, item)
		}
	}
	hist, _ := a.Store.HeistHistory(uid, 8)
	left := domain.HeistRoundSec - int(a.Store.HeistRoundAgeSec(h.ID))
	if left < 0 || h.Status != "running" {
		left = 0
	}
	fillIn := 0
	if h.Status == "open" {
		fillIn = domain.HeistBotWaitSec - int(a.Store.HeistAgeSec(h.ID))
		if fillIn < 0 {
			fillIn = 0
		}
	}
	out["heist"] = map[string]any{
		"round_left": left, "action_sec": domain.HeistRoundSec, "bot_in": fillIn,
		"rounds": domain.HeistRoundsFor(domain.ShopGrade(h.Grade)),
		"id":     h.ID, "grade": h.Grade, "entry": h.Entry, "pot": h.Pot, "progress": h.Progress,
		"target": h.Target, "round": h.Round, "status": h.Status,
		"seats": len(seats), "need": domain.HeistSeats,
	}
	out["me"] = mine
	out["others"] = others
	out["history"] = hist
	writeJSON(w, 200, out)
	return nil
}

// heistJoin: POST /api/heist/join {grade}
func (a *API) heistJoin(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Grade int `json:"grade"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	g := domain.ShopGrade(body.Grade)
	if g < domain.KiloGrade || g > domain.WindowGrade {
		return errors.New("沒有這個檔位")
	}
	if cur, _ := a.Store.MyHeist(uid); cur != nil {
		return errors.New("你已經在桌上了")
	}
	entry := domain.HeistEntry(g)
	var heistID int
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if _, err := store.UpdateChipsTx(tx, uid, -entry); err != nil {
			return err
		}
		h, err := a.Store.OpenHeistTx(tx, int(g))
		if err != nil {
			return err
		}
		heistID = h.ID
		if err := a.Store.JoinHeistTx(tx, h.ID, uid, entry); err != nil {
			return err
		}
		// 注意：MaxOpenConns(1)，tx 裡只能用 tx 自己的查詢（呼叫 store 的非 tx 方法會死鎖）
		var seated int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM heist_seats WHERE heist_id=?`, h.ID).Scan(&seated); err != nil {
			return err
		}
		pot := int(float64(entry*seated) * domain.HeistPotBonusB(g))
		if _, err := tx.Exec(`UPDATE heists SET pot=? WHERE id=?`, pot, h.ID); err != nil {
			return err
		}
		if seated >= domain.HeistSeats {
			if _, err := tx.Exec(`UPDATE heists SET status='running' WHERE id=?`, h.ID); err != nil {
				return err
			}
		}
		return tx.QueryRow(`SELECT chips FROM users WHERE id=?`, uid).Scan(&bal)
	}); err != nil {
		return err
	}
	h, _ := a.Store.HeistByID(heistID)
	seats, _ := a.Store.HeistSeats(heistID)
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal, "heist_id": heistID,
		"seats": len(seats), "started": h != nil && h.Status == "running",
		"message": fmt.Sprintf("已入場（付 %d），%d/%d 人", entry, len(seats), domain.HeistSeats)})
	return nil
}

// heistAct: POST /api/heist/act {action, target}
func (a *API) heistAct(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Action string `json:"action"`
		Target int    `json:"target"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	switch body.Action {
	case domain.HeistCooperate, domain.HeistBetray:
	default:
		return errors.New("要選合作、警戒或背叛")
	}
	h, err := a.Store.MyHeist(uid)
	if err != nil || h == nil {
		return errors.New("你不在任何一桌")
	}
	if h.Status != "running" {
		return errors.New("還沒湊滿 4 人喵")
	}
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		return err
	}
	var me store.HeistSeatRow
	alive := []store.HeistSeatRow{}
	for _, s := range seats {
		if s.UserID == uid {
			me = s
		}
		if s.Alive {
			alive = append(alive, s)
		}
	}
	if !me.Alive {
		return errors.New("你已經出局了")
	}
	if body.Action == domain.HeistBetray {
		valid := false
		for _, s := range alive {
			if s.UserID == body.Target && s.UserID != uid {
				valid = true
			}
		}
		if !valid {
			return errors.New("要選一個活著的對手")
		}
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		return a.Store.SetHeistActionTx(tx, h.ID, uid, body.Action, body.Target)
	}); err != nil {
		return err
	}
	// 全部活人都出手了就結算這一輪
	acted := 0
	for _, s := range alive {
		if s.UserID == uid || s.Action != "" {
			acted++
		}
	}
	if acted >= len(alive) {
		if err := a.resolveHeistRound(h); err != nil {
			return err
		}
	}
	// 逾時結算：有人拖太久就自動幫他算「合作」，讓回合往前走
	if err := a.heistTimeout(h); err != nil {
		return err
	}
	// bot 也出手（這樣單人也能馬上看到結果）
	if err := a.botVotes(h); err != nil {
		return err
	}
	seats, _ = a.Store.HeistSeats(h.ID)
	aliveN, actedN := 0, 0
	for _, s := range seats {
		if !s.Alive {
			continue
		}
		aliveN++
		if s.Action != "" {
			actedN++
		}
	}
	if actedN >= aliveN {
		if err := a.resolveHeistRound(h); err != nil {
			return err
		}
	}
	writeJSON(w, 200, map[string]any{"ok": true, "acted": aliveN - actedN,
		"message": "已出手，等其他人…"})
	return nil
}

// heistTimeout: 一輪超過 30 秒還沒全員出手 → 沒出手的人自動算「合作」並結算。
// 這樣只要有一個人掛機，其他人也不會被卡住。
func (a *API) heistTimeout(h *store.Heist) error {
	if h.Status != "running" {
		return nil
	}
	age := a.Store.HeistRoundAgeSec(h.ID)
	if age < float64(domain.HeistRoundSec) {
		return nil
	}
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		return err
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		for _, s := range seats {
			if s.Alive && s.Action == "" {
				if _, err := tx.Exec(`UPDATE heist_seats SET action='cooperate', target=0 WHERE heist_id=? AND user_id=?`,
					h.ID, s.UserID); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return a.resolveHeistRound(h)
}

// maybeFillBots: 桌子開著超過 5 分鐘、人還不滿 → 補 bot 開局。
func (a *API) maybeFillBots(h *store.Heist) error {
	if h.Status != "open" {
		return nil
	}
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		return err
	}
	if len(seats) >= domain.HeistSeats {
		return nil
	}
	return a.Store.WithTx(func(tx *store.Tx) error {
		var age float64
		if err := tx.QueryRow(`SELECT (julianday('now') - julianday(created_at)) * 86400.0 FROM heists WHERE id=?`, h.ID).Scan(&age); err != nil {
			return err
		}
		if age < float64(domain.HeistBotWaitSec) {
			return nil // 再等等，說不定真人要進來
		}
		bots := domain.PickHeistBots(domain.HeistSeats-len(seats), domain.RandSource)
		if _, err := a.Store.FillHeistBotsTx(tx, h.ID, h.Entry, domain.HeistSeats, bots); err != nil {
			return err
		}
		// 補滿後獎池就是完整的一份（寶石價值 = 4 × 入場費 × 1.5）
		pot := int(float64(domain.HeistEntry(domain.ShopGrade(h.Grade))*domain.HeistSeats) * domain.HeistPotMul)
		_, err = tx.Exec(`UPDATE heists SET pot=? WHERE id=?`, pot, h.ID)
		return err
	})
}

// botVotes: 讓桌上還沒出手的 bot 按自己的脾氣投票。
func (a *API) botVotes(h *store.Heist) error {
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		return err
	}
	bots, err := a.Store.HeistBotsIn(h.ID)
	if err != nil || len(bots) == 0 {
		return err
	}
	alive := []store.HeistSeatRow{}
	for _, s := range seats {
		if s.Alive {
			alive = append(alive, s)
		}
	}
	return a.Store.WithTx(func(tx *store.Tx) error {
		for _, s := range alive {
			b, ok := bots[s.UserID]
			if !ok || s.Action != "" {
				continue
			}
			others := []domain.HeistSeat{}
			for _, o := range alive {
				if o.UserID != s.UserID {
					others = append(others, domain.HeistSeat{UserID: o.UserID, Name: o.Name, Alive: true, Entry: o.Entry})
				}
			}
			me := domain.HeistSeat{UserID: s.UserID, Name: s.Name, Alive: true, Entry: s.Entry}
			act, target := domain.HeistBotVote(b, me, s.Exposed, others, h.Progress, h.Target, domain.RandSource)
			if _, err := tx.Exec(`UPDATE heist_seats SET action=?, target=? WHERE heist_id=? AND user_id=?`,
				act, target, h.ID, s.UserID); err != nil {
				return err
			}
		}
		return nil
	})
}

// resolveHeistRound: 結算一輪（並在條件達成時結算整桌）。
func (a *API) resolveHeistRound(h *store.Heist) error {
	seats, err := a.Store.HeistSeats(h.ID)
	if err != nil {
		return err
	}
	ds := make([]domain.HeistSeat, 0, len(seats))
	for _, s := range seats {
		ds = append(ds, domain.HeistSeat{UserID: s.UserID, Name: s.Name, Alive: s.Alive,
			Action: s.Action, Target: s.Target, Entry: s.Entry})
	}
	res := domain.ResolveHeistRound(ds, h.Progress, h.Target, domain.RandSource)
	round := h.Round + 1
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if err := a.Store.ApplyHeistRoundTx(tx, h.ID, res.Progress, round, res); err != nil {
			return err
		}
		return a.Store.ResetHeistActionsTx(tx, h.ID)
	}); err != nil {
		return err
	}
	// 結算條件：進度滿（倖存者平分）/ 只剩一人（獨吞）/ 崩塌（沒收）/ 回合用完（沒收）
	alive := []int{}
	for _, s := range seats {
		if s.Alive && res.Deaths[s.UserID] == 0 {
			alive = append(alive, s.UserID)
		}
	}
	done := res.Collapse || res.Progress >= h.Target || len(alive) <= 1 || round >= domain.HeistRoundsFor(domain.ShopGrade(h.Grade))
	if !done {
		return nil
	}
	payout := map[int]int{}
	switch {
	case res.Collapse || len(alive) == 0: // 崩塌／全滅：獎池沒收
		payout = map[int]int{}
	case len(alive) == 1: // 獨吞
		payout = domain.HeistPayout(domain.ShopGrade(h.Grade), h.Pot, alive)
	case res.Progress >= h.Target: // 一起挖到
		payout = domain.HeistPayout(domain.ShopGrade(h.Grade), h.Pot, alive)
	default: // 5 輪用完還沒挖到：入場費沒收
		payout = map[int]int{}
	}
	return a.Store.WithTx(func(tx *store.Tx) error {
		// 獎池按進度折算（用戶 2026-09-16 定案）：沒挖到就拿不到滿額。
		// 例：進度 4/18 只剩一人 → 只拿 4/18 的獎池，不再是「沒挖到卻獨吞全部」。
		if len(payout) > 0 && h.Target > 0 && res.Progress < h.Target {
			scale := float64(res.Progress) / float64(h.Target)
			for uid, amt := range payout {
				payout[uid] = int(math.Round(float64(amt) * scale))
			}
		}
		return a.Store.FinishHeistTx(tx, h.ID, payout)
	})
}

// heistFill: POST /api/heist/fill —— 不想等，直接補 bot 開局
func (a *API) heistFill(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	h, err := a.Store.MyHeist(uid)
	if err != nil || h == nil {
		return errors.New("你不在任何一桌")
	}
	if h.Status != "open" {
		return errors.New("這桌已經開局了")
	}
	var added int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		var have int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM heist_seats WHERE heist_id=?`, h.ID).Scan(&have); err != nil {
			return err
		}
		if have >= domain.HeistSeats {
			return nil
		}
		bots := domain.PickHeistBots(domain.HeistSeats-have, domain.RandSource)
		n, err := a.Store.FillHeistBotsTx(tx, h.ID, h.Entry, domain.HeistSeats, bots)
		added = n
		if err != nil {
			return err
		}
		pot := int(float64(domain.HeistEntry(domain.ShopGrade(h.Grade))*domain.HeistSeats) * domain.HeistPotMul)
		_, err = tx.Exec(`UPDATE heists SET pot=? WHERE id=?`, pot, h.ID)
		return err
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "added": added,
		"message": fmt.Sprintf("已補 %d 位 bot，開局！", added)})
	return nil
}

// heistLeave: POST /api/heist/leave
func (a *API) heistLeave(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	h, err := a.Store.MyHeist(uid)
	if err != nil {
		return err
	}
	if h == nil {
		// 結束的桌子也要能離開（不然按鈕只會回「你不在任何一桌」＝卡在結算畫面）
		rec, err := a.Store.MyHeistRecent(uid, 1440)
		if err != nil || rec == nil {
			return errors.New("你不在任何一桌")
		}
		if err := a.Store.MarkHeistSeatLeft(rec.ID, uid); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"ok": true, "message": "已離開這桌（紀錄留在「我的奪寶紀錄」）"})
		return nil
	}
	if h.Status != "open" {
		return errors.New("這桌已經開局或結束了，不能退費")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if _, err := tx.Exec(`DELETE FROM heist_seats WHERE heist_id=? AND user_id=?`, h.ID, uid); err != nil {
			return err
		}
		var remaining int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM heist_seats WHERE heist_id=?`, h.ID).Scan(&remaining); err != nil {
			return err
		}
		pot := int(float64(h.Entry*remaining) * domain.HeistPotBonusB(domain.ShopGrade(h.Grade)))
		if _, err := tx.Exec(`UPDATE heists SET pot=? WHERE id=?`, pot, h.ID); err != nil {
			return err
		}
		b, err := store.UpdateChipsTx(tx, uid, h.Entry)
		if err != nil {
			return err
		}
		bal = b
		return nil
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal, "message": "已退出，入場費退回"})
	return nil
}
