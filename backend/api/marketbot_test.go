package api_test

import (
	"testing"
)

// 放太久的掛單會被拍賣 bot 收走，賣家拿得到錢（95%）。
func TestMarketBotBuysStaleListing(t *testing.T) {
	srv, st := setup(t)
	alice := newClient(t, srv, "botseller")

	// 買一顆最便宜的公斤料並掛一個低價（低於任何 bot 的出價上限）
	shop := alice.do("GET", "/api/shop", nil)
	items := shop["grades"].([]any)[0].(map[string]any)["items"].([]any)
	stoneID := items[0].(map[string]any)["id"].(string)
	alice.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})
	alice.do("POST", "/api/market/list", map[string]any{"stone_id": stoneID, "ask_price": 1})

	before := int(alice.do("GET", "/api/me", nil)["chips"].(float64))

	// 剛掛上去、還是新的 → bot 先不動手（真人優先）
	alice.do("GET", "/api/market", nil)
	mkt := alice.do("GET", "/api/market", nil)
	found := false
	for _, l := range mkt["listings"].([]any) {
		if l.(map[string]any)["stone_id"] == stoneID {
			found = true
		}
	}
	if !found {
		t.Fatal("剛掛的單就被 bot 收走了（應該先留給真人買）")
	}

	// 把它往前挪 5 小時，再看一次市場
	listings := alice.do("GET", "/api/market", nil)["listings"].([]any)
	var lid int
	for _, l := range listings {
		m := l.(map[string]any)
		if m["stone_id"] == stoneID {
			lid = int(m["id"].(float64))
		}
	}
	if lid == 0 {
		t.Fatal("找不到剛才的掛單")
	}
	if err := st.BackdateListing(lid, 5); err != nil {
		t.Fatal(err)
	}
	alice.do("GET", "/api/market", nil)

	// 石頭不該還在市場上
	for _, l := range alice.do("GET", "/api/market", nil)["listings"].([]any) {
		if l.(map[string]any)["stone_id"] == stoneID {
			t.Fatal("放 5 小時的單還沒被 bot 收走")
		}
	}
	after := int(alice.do("GET", "/api/me", nil)["chips"].(float64))
	if after <= before {
		t.Fatalf("賣家沒收到錢: before=%d after=%d", before, after)
	}
	// 市場要看得到 bot 收料紀錄
	buys := alice.do("GET", "/api/market", nil)["bot_buys"].([]any)
	if len(buys) == 0 {
		t.Fatal("市場沒有 bot 收料紀錄")
	}
	t.Logf("bot 收走 %s，賣家 +%d 籌碼，紀錄: %v", stoneID, after-before, buys[0])
}
