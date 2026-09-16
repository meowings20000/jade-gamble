package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"jade-gamble/backend/store"
)

// 轉賬（2026-09-16 新增）
//
// 規則：
//   - 不填條件 → 即時到賬（sent）。
//   - 填了條件 → 先扣發起人的籌碼託管，等對方接受（accepted）才入賬；
//     對方拒絕（declined）或發起人自己取消（cancelled）→ 全額退還。
//   - 不能轉給自己；金額必須 > 0 且不超過自己手上的籌碼。
type transferReq struct {
	To        string `json:"to"`
	Amount    int    `json:"amount"`
	Condition string `json:"condition"`
}

type transferActReq struct {
	ID int `json:"id"`
}

// transferSend: POST /api/transfer
func (a *API) transferSend(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body transferReq
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	body.To = strings.TrimSpace(body.To)
	body.Condition = strings.TrimSpace(body.Condition)
	if body.To == "" {
		return errors.New("請填對方名稱")
	}
	if body.Amount <= 0 {
		return errors.New("金額必須大於 0")
	}
	if len([]rune(body.Condition)) > 120 {
		return errors.New("條件太長了（最多 120 字）")
	}
	to, err := a.Store.UserByUsername(body.To)
	if err != nil {
		return errors.New("找不到這個玩家（名稱要完全一樣）")
	}
	if to.ID == uid {
		return errors.New("不能轉給自己")
	}

	status := store.TransferSent
	if body.Condition != "" {
		status = store.TransferPending
	}

	var myChips int
	var transferID int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		bal, err := store.UpdateChipsTx(tx, uid, -body.Amount)
		if err != nil {
			return err
		}
		myChips = bal
		if status == store.TransferSent {
			if _, err := store.UpdateChipsTx(tx, to.ID, body.Amount); err != nil {
				return err
			}
		}
		id, err := a.Store.CreateTransferTx(tx, uid, to.ID, body.Amount, body.Condition, status)
		if err != nil {
			return err
		}
		transferID = id
		return nil
	}); err != nil {
		return err
	}

	msg := fmt.Sprintf("已轉 %d 籌碼給 %s", body.Amount, to.Username)
	if status == store.TransferPending {
		msg = fmt.Sprintf("已送出（附條件），等 %s 接受", to.Username)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": msg, "chips": myChips,
		"transfer_id": transferID, "status": status})
	return nil
}

// transfers: GET /api/transfers
func (a *API) transfers(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	incoming, outgoing, history, err := a.Store.ListTransfers(uid, 12)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"incoming": incoming, "outgoing": outgoing, "history": history})
	return nil
}

// transferAccept: POST /api/transfer/accept —— 接受別人的轉賬（有條件的才需要）。
func (a *API) transferAccept(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body transferActReq
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	t, err := a.Store.TransferByID(body.ID)
	if err != nil {
		return errors.New("找不到這筆轉賬")
	}
	if t.Status != store.TransferPending {
		return errors.New("這筆轉賬已經處理過了")
	}
	if t.ToID != uid {
		return errors.New("這筆轉賬不是給你的")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, t.Amount)
		if err != nil {
			return err
		}
		bal = b
		return a.Store.ResolveTransferTx(tx, t.ID, store.TransferAccepted)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal,
		"message": fmt.Sprintf("已接受 %s 的 %d 籌碼", t.FromName, t.Amount)})
	return nil
}

// transferDecline: POST /api/transfer/decline —— 拒絕，籌碼退還發起人。
func (a *API) transferDecline(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body transferActReq
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	t, err := a.Store.TransferByID(body.ID)
	if err != nil {
		return errors.New("找不到這筆轉賬")
	}
	if t.Status != store.TransferPending {
		return errors.New("這筆轉賬已經處理過了")
	}
	if t.ToID != uid {
		return errors.New("這筆轉賬不是給你的")
	}
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		if _, err := store.UpdateChipsTx(tx, t.FromID, t.Amount); err != nil {
			return err
		}
		return a.Store.ResolveTransferTx(tx, t.ID, store.TransferDeclined)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true,
		"message": fmt.Sprintf("已拒絕，籌碼退還 %s", t.FromName)})
	return nil
}

// transferCancel: POST /api/transfer/cancel —— 發起人自己收回。
func (a *API) transferCancel(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body transferActReq
	if err := readJSON(w, r, &body); err != nil {
		return errors.New("參數錯誤")
	}
	t, err := a.Store.TransferByID(body.ID)
	if err != nil {
		return errors.New("找不到這筆轉賬")
	}
	if t.Status != store.TransferPending {
		return errors.New("這筆轉賬已經處理過了")
	}
	if t.FromID != uid {
		return errors.New("這筆轉賬不是你發起的")
	}
	var bal int
	if err := a.Store.WithTx(func(tx *store.Tx) error {
		b, err := store.UpdateChipsTx(tx, uid, t.Amount)
		if err != nil {
			return err
		}
		bal = b
		return a.Store.ResolveTransferTx(tx, t.ID, store.TransferCancelled)
	}); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "chips": bal,
		"message": fmt.Sprintf("已取消，%s 籌碼退回", strconv.Itoa(t.Amount))})
	return nil
}
