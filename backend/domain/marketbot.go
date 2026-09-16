package domain

// 拍賣機器人（收料 bot）—— 沒人標的料，5 分鐘後就有 bot 來收。
//
// 每個 bot 用自己的「眼力」估價：出價上限 = 眼力 × 真值（BaseValue）。
// 眼力高的走漏眼（願意用高於真值的價錢收），眼力低的精明（只撿便宜）。
// 十隻的眼力平均 ≈ 0.88（< 1），所以長期把料丟給 bot 收一定虧，
// 「想賺」還是得自己切／磨／刮或賣給真人。
//
// 沒人理的掛單 5 分鐘後開始有 bot 出手（StickyMins）；每隻一次收幾件（Appetite）
// 與眼力都不同，所以市場上不會一次被同一隻清空。

// MarketBot: 一隻收料機器人。
type MarketBot struct {
	Key        string
	Name       string
	Eye        float64  // 出價上限 = Eye × 真值
	StickyMins float64  // 放超過這麼久（分鐘）才出手
	Appetite   int      // 每次最多收幾件
	Quips      []string // 收料時留給賣家的話
}

// BotStickyMins: 沒人理的掛單幾分鐘後開始有 bot 收。
const BotStickyMins = 5

// MarketBots: 十隻，眼力由闊到精（1.30 → 0.55）。
var MarketBots = []MarketBot{
	{Key: "sanjiao", Name: "三腳貓", Eye: 1.30, StickyMins: 5, Appetite: 2,
		Quips: []string{"這塊我看好！先收了。", "皮殼有意思，賭一把。", "手快有手慢無，成交！"}},
	{Key: "shoumai", Name: "收買佬", Eye: 1.15, StickyMins: 5, Appetite: 2,
		Quips: []string{"價錢合理，我要了。", "放這麼久沒人要？那我收。", "剛好缺這尺寸。"}},
	{Key: "bantong", Name: "半桶水", Eye: 1.05, StickyMins: 5, Appetite: 2,
		Quips: []string{"看得出點門道，收了。", "應該有肉，試試手氣。"}},
	{Key: "laojiefang", Name: "老街坊", Eye: 0.98, StickyMins: 5, Appetite: 3,
		Quips: []string{"照市價收，不賺你。", "街坊生意，公道。"}},
	{Key: "tiepan", Name: "鐵算盤", Eye: 0.90, StickyMins: 5, Appetite: 3,
		Quips: []string{"這個價才划算。", "算過了，收。", "再高一分我都不出。"}},
	{Key: "jianlou", Name: "撿漏王", Eye: 0.85, StickyMins: 5, Appetite: 3,
		Quips: []string{"有漏可撿，別聲張。", "低價進，慢慢磨。"}},
	{Key: "maiyusheng", Name: "賣魚勝", Eye: 0.78, StickyMins: 5, Appetite: 4,
		Quips: []string{"當魚腩收，賣唔出就擺景。", "平啲得唔得？得，我要。"}},
	{Key: "guijian", Name: "鬼見愁", Eye: 0.68, StickyMins: 5, Appetite: 4,
		Quips: []string{"清倉價，勉強收。", "當廢料收，別嫌少。", "這價錢你也不虧，散了吧。"}},
	{Key: "dazhonglian", Name: "打腫臉", Eye: 0.60, StickyMins: 5, Appetite: 5,
		Quips: []string{"先收著，回頭再算。", "撐場面，收了。"}},
	{Key: "shoupolang", Name: "收破爛", Eye: 0.55, StickyMins: 5, Appetite: 6,
		Quips: []string{"當石頭秤，一斤幾錢？", "破爛價，賣就賣。"}},
}

// BotMaxPrice: 這隻 bot 對這顆石頭最多願意出多少。
func BotMaxPrice(b MarketBot, st *Stone) int {
	return int(float64(st.BaseValue()) * b.Eye)
}

// BotBuys: bot 會不會收這張單（喊價在牠的出價上限之內）。
func BotBuys(b MarketBot, st *Stone, askPrice int) bool {
	return askPrice <= BotMaxPrice(b, st)
}

// MarketBotByKey: 依 key 取 bot。
func MarketBotByKey(key string) (MarketBot, bool) {
	for _, b := range MarketBots {
		if b.Key == key {
			return b, true
		}
	}
	return MarketBot{}, false
}

// BotQuip: 這隻 bot 收料時會留的一句話。
func BotQuip(b MarketBot, r Rand) string {
	if len(b.Quips) == 0 {
		return ""
	}
	return b.Quips[r.Intn(len(b.Quips))]
}
