package domain

import (
	"testing"
)

// 拍賣 bot: 眼力決定出價；走漏眼的收得貴，精明的只撿便宜。
func TestMarketBotsEye(t *testing.T) {
	st := &Stone{Price: 10000, Quality: OilGreen, Variety: Base} // 真值 ≈ 10,200
	base := st.BaseValue()
	if base <= 0 {
		t.Fatal("bad fixture")
	}
	ming, _ := MarketBotByKey("sanjiao") // 走漏眼 1.30×
	zhou, _ := MarketBotByKey("guijian") // 精明 0.68×
	chen, _ := MarketBotByKey("tiepan")  // 0.90×

	// 喊價高於真值：只有走漏眼的肯收
	ask := int(float64(base) * 1.20)
	if !BotBuys(ming, st, ask) {
		t.Error("三腳貓應該走漏眼、肯用高於真值的價收")
	}
	if BotBuys(chen, st, ask) || BotBuys(zhou, st, ask) {
		t.Error("精明的 bot 不該收高於真值的單")
	}
	// 喊價低於真值：大家都收
	low := int(float64(base) * 0.60)
	for _, b := range []struct {
		name string
		ok   bool
	}{{ming.Name, BotBuys(ming, st, low)}, {chen.Name, BotBuys(chen, st, low)}, {zhou.Name, BotBuys(zhou, st, low)}} {
		if !b.ok {
			t.Errorf("%s 連便宜貨都不收", b.name)
		}
	}
	// 出價上限排序：阿明 > 阿強 > 陳師傅 > 老周
	order := []string{"sanjiao", "shoumai", "bantong", "laojiefang", "tiepan", "jianlou", "maiyusheng", "guijian", "dazhonglian", "shoupolang"}
	prev := 1e18
	for _, k := range order {
		b, _ := MarketBotByKey(k)
		max := float64(BotMaxPrice(b, st))
		if max >= prev {
			t.Errorf("%s 的出價上限 %.0f 沒有比前一隻低（前 %.0f）", b.Name, max, prev)
		}
		prev = max
	}
	// 四隻平均眼力要略低於 1（bot 不會無腦送錢）
	sum := 0.0
	for _, b := range MarketBots {
		sum += b.Eye
	}
	if avg := sum / float64(len(MarketBots)); avg > 1.0 {
		t.Errorf("bot 平均眼力 %.2f > 1，會被套利", avg)
	}
	// 十隻 bot，沒人理的掛單 5 分鐘後開始有 bot 收
	if len(MarketBots) != 10 {
		t.Errorf("bot 應該 10 隻，得到 %d", len(MarketBots))
	}
	for _, b := range MarketBots {
		if b.StickyMins != BotStickyMins {
			t.Errorf("%s 的耐心 %.1f 分鐘，應該統一 %v 分鐘", b.Name, b.StickyMins, BotStickyMins)
		}
		if b.Appetite <= 0 || len(b.Quips) == 0 {
			t.Errorf("%s 沒有食量或留言", b.Name)
		}
	}
}
