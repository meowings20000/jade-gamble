package api

// Claude 共產池 API（2026-10-05）
//
// GET  /api/ai/pool          池子狀態（登入才看得到 my_contrib；匿名也行）
// POST /api/ai/pool/contribute {amount} 捐喵喵幣進公共池
//
// 以下給 DMIT 守門員（cron）用，secret 放 .env，不進 repo：
// POST /api/ai/pool/sync     {five_hour_util,...} 回報用量 → 回傳 {unlock,lock}

import (
	"errors"
	"net/http"
	"time"

	"jade-gamble/backend/store"
)

// aiPoolView: GET /api/ai/pool
func (a *API) aiPoolView(w http.ResponseWriter, r *http.Request) error {
	uid, _ := a.userID(r) // 匿名允許（uid=0）
	v, err := a.Store.AIView(uid)
	if err != nil {
		return err
	}
	board, berr := a.Store.AIContribList(20)
	if berr != nil {
		board = []map[string]any{{"_err": berr.Error()}}
	}
	writeJSON(w, 200, map[string]any{"ok": true, "pool": v, "board": board})
	return nil
}

// aiPoolContribute: POST /api/ai/pool/contribute
func (a *API) aiPoolContribute(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := readJSON(w, r, &body); err != nil {
		return err
	}
	if body.Amount <= 0 || body.Amount > 100_000_000 {
		return errors.New("金額需在 1 ~ 100,000,000 之間")
	}
	var view store.AIPoolView
	var chipsAfter int64
	err = a.Store.WithTx(func(tx *store.Tx) error {
		_, chips, e := a.Store.AIContributeTx(tx, uid, body.Amount)
		if e != nil {
			return e
		}
		chipsAfter = chips
		return nil
	})
	if err != nil {
		return err
	}
	// 交易外再查 pool 狀態（ 避免 SQLite 單寫者連線死鎖）
	v, err := a.Store.AIView(uid)
	if err != nil {
		return err
	}
	view = v
	view.ChipsAfter = chipsAfter
	writeJSON(w, 200, map[string]any{"ok": true, "pool": view})
	return nil
}

// aiPoolOpen: POST /api/ai/pool/open —— ★ 開市按鈕（池子達標才有效）
// 開市 1 小時：pool 清零、open_until=now+1h；DMIT 守門員依 unlock 指令開渠道
func (a *API) aiPoolOpen(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	var body struct{}
	_ = readJSON(w, r, &body) // body 可空
	v, err := a.Store.AIMarketOpen(uid, time.Now())
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"ok": true, "pool": v})
	return nil
}

// aiPoolQuick20: GET /api/ai/pool/quick20 —— 「捐池子 20%」按鈕的可捐額
func (a *API) aiPoolQuick20(w http.ResponseWriter, r *http.Request) error {
	uid, err := a.userID(r)
	if err != nil {
		return err
	}
	q := a.Store.Quick20(uid)
	writeJSON(w, 200, map[string]any{"ok": true, "target": q.Target, "my": q.My,
		"can": q.Can, "available": q.Can > 0, "chips": q.Chips})
	return nil
}

// aiPoolSync: POST /api/ai/pool/sync（DMIT 守門員）
func (a *API) aiPoolSync(w http.ResponseWriter, r *http.Request) error {
	// secret 驗證：X-AI-Sync-Token
	tok := r.Header.Get("X-AI-Sync-Token")
	if a.AISyncToken == "" || tok == "" || tok != a.AISyncToken {
		writeErr(w, 401, "forbidden")
		return nil
	}
	var snap store.AIUsageSnapshot
	if err := readJSON(w, r, &snap); err != nil {
		return err
	}
	view, unlock, lock, err := a.Store.AISync(snap, time.Now())
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{
		"ok": true, "pool": view, "unlock": unlock, "lock": lock,
	})
	return nil
}

// aiPoolDebug: GET /api/ai/_debug（暫時）
func (a *API) aiPoolDebug(w http.ResponseWriter, r *http.Request) error {
	contrib, err1 := a.Store.DebugAIContrib()
	board, err2 := a.Store.AIContribList(20)
	writeJSON(w, 200, map[string]any{
		"ok": true, "contrib_4col": contrib, "board_5col_join": board,
		"err1": fmtErr(err1), "err2": fmtErr(err2),
	})
	return nil
}

func fmtErr(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
