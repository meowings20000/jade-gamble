package api

import (
	"errors"
	"fmt"
	"net/http"

	"jade-gamble/backend/store"
)

// 獎勵兌換 API（2026-09-16）
//
// GET  /api/rewards          我的申請
// POST /api/rewards/request  {title, note, cost} 送審（不立刻扣籌碼）
// GET  /api/admin/rewards    待審清單（管理員）
// POST /api/admin/rewards/decide {id, approve, note} 通過時才扣籌碼

const RewardMaxCost = 2000000

func (a *API) rewardsList(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	list, err := a.Store.RewardRequests(uid, 20)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"requests": list,
		"presets": []map[string]any{
			{"title": "一百萬大獎", "cost": 1000000, "note": "累積到 100 萬籌碼可申請，內容由站長親自安排"},
			{"title": "自訂獎勵", "cost": 0, "note": "自己許願（要多少籌碼、想要什麼），站長審核後決定"},
		}})
	return nil
}

func (a *API) rewardsRequest(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Title string `json:"title"`
		Note  string `json:"note"`
		Cost  int    `json:"cost"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	if len([]rune(body.Title)) < 2 {
		return errors.New("請寫獎勵名稱")
	}
	if len([]rune(body.Title)) > 40 || len([]rune(body.Note)) > 200 {
		return errors.New("名稱 40 字、說明 200 字以內")
	}
	if body.Cost < 0 || body.Cost > RewardMaxCost {
		return errors.New("籌碼金額不合理")
	}
	var id int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		var err error
		id, err = a.Store.CreateRewardTx(tx, uid, body.Title, body.Note, body.Cost)
		return err
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": id,
		"message": "已送出申請，等站長審核"})
	return nil
}

// adminRewards: GET /api/admin/rewards
func (a *API) adminRewards(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	if !a.isAdmin(uid) {
		return errors.New("只有管理員可以進來")
	}
	list, err := a.Store.RewardRequests(0, 60)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"requests": list})
	return nil
}

// adminRewardDecide: POST /api/admin/rewards/decide {id, approve, note}
func (a *API) adminRewardDecide(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	if !a.isAdmin(uid) {
		return errors.New("只有管理員可以進來")
	}
	var body struct {
		ID      int    `json:"id"`
		Approve bool   `json:"approve"`
		Note    string `json:"note"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	req, err := a.Store.RewardByID(body.ID)
	if err != nil || req == nil {
		return errors.New("找不到這筆申請")
	}
	if req.Status != "pending" {
		return errors.New("這筆已經處理過了")
	}
	status := "rejected"
	if body.Approve {
		status = "approved"
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if body.Approve && req.Cost > 0 {
			var chips int
			if err := tx.QueryRow(`SELECT chips FROM users WHERE id=?`, req.UserID).Scan(&chips); err != nil {
				return err
			}
			if chips < req.Cost {
				return errors.New("玩家籌碼不夠，無法扣款")
			}
			if _, err := store.UpdateChipsTx(tx, req.UserID, -req.Cost); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE reward_requests SET status=?, admin_note=?, decided_at=datetime('now')
			WHERE id=?`, status, body.Note, body.ID); err != nil {
			return err
		}
		_, err := tx.Exec(`INSERT INTO admin_log (admin_id, action, target_id, amount, note) VALUES (?,?,?,?,?)`,
			uid, "reward_"+status, req.UserID, req.Cost, fmt.Sprintf("%s #%d", body.Note, body.ID))
		return err
	}); err != nil {
		return err
	}
	msg := "已拒絕"
	if body.Approve {
		msg = "已通過"
	}
	writeJSON(w, 200, map[string]any{"ok": true, "status": status, "message": msg})
	return nil
}
