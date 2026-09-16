package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// doErr: 預期失敗的請求（不會 Fatal），回 status + body。
func (c *client) doErr(method, path string, body interface{}) (int, map[string]any) {
	c.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.srv.URL+path, rdr)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Cookie", c.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func chips(t *testing.T, c *client) int {
	t.Helper()
	return int(c.do("GET", "/api/me", nil)["chips"].(float64))
}

// 無條件轉賬：即時到賬。
func TestTransferImmediate(t *testing.T) {
	srv, _ := setup(t)
	alex := newClient(t, srv, "alex")
	bob := newClient(t, srv, "bob")

	a0, b0 := chips(t, alex), chips(t, bob)
	res := alex.do("POST", "/api/transfer", map[string]any{"to": "bob", "amount": 8000})
	if chips(t, alex) != a0-8000 {
		t.Fatalf("alex 沒扣錢: %d -> %d", a0, chips(t, alex))
	}
	if chips(t, bob) != b0+8000 {
		t.Fatalf("bob 沒收到錢: %d -> %d", b0, chips(t, bob))
	}
	if res["status"] != "sent" {
		t.Fatalf("無條件應該是即時到賬，得到 %v", res["status"])
	}

	// 歷史裡看得到
	hist := bob.do("GET", "/api/transfers", nil)["history"].([]any)
	if len(hist) == 0 {
		t.Fatal("轉賬沒有出現在歷史")
	}
	h := hist[0].(map[string]any)
	if h["from"] != "alex" || h["to"] != "bob" || h["status"] != "sent" {
		t.Fatalf("歷史內容不對: %v", h)
	}
}

// 附條件轉賬：先扣錢託管，等對方接受才入賬。
func TestTransferWithConditionRequiresAccept(t *testing.T) {
	srv, _ := setup(t)
	alex := newClient(t, srv, "alex")
	bob := newClient(t, srv, "bob")

	a0, b0 := chips(t, alex), chips(t, bob)
	cond := "幫我把倉庫那顆公斤料磨到不可能再磨"
	res := alex.do("POST", "/api/transfer", map[string]any{"to": "bob", "amount": 5000, "condition": cond})
	if res["status"] != "pending" {
		t.Fatalf("附條件應該待接受，得到 %v", res["status"])
	}
	if chips(t, alex) != a0-5000 {
		t.Fatalf("附條件時應該先扣錢託管: %d -> %d", a0, chips(t, alex))
	}
	if chips(t, bob) != b0 {
		t.Fatalf("對方還沒接受就不該入賬: %d -> %d", b0, chips(t, bob))
	}

	// 收件人看得到條件
	inc := bob.do("GET", "/api/transfers", nil)["incoming"].([]any)
	if len(inc) != 1 {
		t.Fatalf("待接受的轉賬數不對: %v", inc)
	}
	tr := inc[0].(map[string]any)
	if tr["condition"] != cond {
		t.Fatalf("條件沒帶到: %v", tr["condition"])
	}
	id := int(tr["id"].(float64))

	// 接受 → 入賬
	bob.do("POST", "/api/transfer/accept", map[string]any{"id": id})
	if chips(t, bob) != b0+5000 {
		t.Fatalf("接受後沒入賬: %d -> %d", b0, chips(t, bob))
	}
	// 不能重複接受
	if code, _ := bob.doErr("POST", "/api/transfer/accept", map[string]any{"id": id}); code < 300 {
		t.Fatal("同一筆轉賬被接受了兩次")
	}
}

// 拒絕 → 全額退還發起人。
func TestTransferDecline(t *testing.T) {
	srv, _ := setup(t)
	alex := newClient(t, srv, "alex")
	bob := newClient(t, srv, "bob")

	a0, b0 := chips(t, alex), chips(t, bob)
	alex.do("POST", "/api/transfer", map[string]any{"to": "bob", "amount": 4000, "condition": "先幫我刮三格"})
	tr := bob.do("GET", "/api/transfers", nil)["incoming"].([]any)[0].(map[string]any)
	bob.do("POST", "/api/transfer/decline", map[string]any{"id": int(tr["id"].(float64))})

	if chips(t, alex) != a0 {
		t.Fatalf("拒絕後沒退還發起人: %d -> %d", a0, chips(t, alex))
	}
	if chips(t, bob) != b0 {
		t.Fatalf("拒絕後對方不該有錢: %d -> %d", b0, chips(t, bob))
	}
}

// 發起人可以自己取消（拿回託管的錢）。
func TestTransferCancel(t *testing.T) {
	srv, _ := setup(t)
	alex := newClient(t, srv, "alex")
	bob := newClient(t, srv, "bob")

	a0 := chips(t, alex)
	out := alex.do("POST", "/api/transfer", map[string]any{"to": "bob", "amount": 6000, "condition": "借我你的石頭看看"})
	alex.do("POST", "/api/transfer/cancel", map[string]any{"id": int(out["transfer_id"].(float64))})
	if chips(t, alex) != a0 {
		t.Fatalf("取消後沒退回: %d -> %d", a0, chips(t, alex))
	}
	// 對方不能替發起人取消
	out2 := alex.do("POST", "/api/transfer", map[string]any{"to": "bob", "amount": 1000, "condition": "x"})
	if code, _ := bob.doErr("POST", "/api/transfer/cancel", map[string]any{"id": int(out2["transfer_id"].(float64))}); code < 300 {
		t.Fatal("收件人竟然能取消別人的轉賬")
	}
}

// 壞輸入。
func TestTransferBadInput(t *testing.T) {
	srv, _ := setup(t)
	alex := newClient(t, srv, "alex")
	cases := []struct {
		name string
		body map[string]any
	}{
		{"轉給自己", map[string]any{"to": "alex", "amount": 100}},
		{"不存在的人", map[string]any{"to": "nobody_at_all", "amount": 100}},
		{"金額 0", map[string]any{"to": "bob", "amount": 0}},
		{"負數金額", map[string]any{"to": "bob", "amount": -500}},
		{"超過持有", map[string]any{"to": "bob", "amount": 999999999}},
		{"沒填對象", map[string]any{"to": "", "amount": 100}},
	}
	newClient(t, srv, "bob")
	for _, c := range cases {
		if code, body := alex.doErr("POST", "/api/transfer", c.body); code < 300 {
			t.Errorf("%s 竟然成功: %v", c.name, body)
		}
	}
	// 壞輸入不該把錢吃掉或生出來
	if got := chips(t, alex); got != 50000 {
		t.Fatalf("失敗的轉賬動到餘額: %d", got)
	}
}
