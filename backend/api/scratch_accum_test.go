package api_test

import (
	"testing"
)

// 回歸測試：刮石必須真的累加。
//
// 2026-09-16 修的正式 bug：reveal 用 Reveal() 回放已開格，但 Reveal 看到
// revealed 已 true 就早退，導致累積值永遠只有最後一格的價值——刮到底只賠付
// 約 1/12 的石頭價值（玩家體感「刮很虧、一下就爆」）。
func TestScratchAccumulates(t *testing.T) {
	srv, _ := setup(t)
	c := newClient(t, srv, "scratch_accum")

	shop := c.do("GET", "/api/shop", nil)
	item := shop["grades"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	stoneID := item["id"].(string)
	c.do("POST", "/api/shop/buy", map[string]string{"stone_id": stoneID})

	c.do("POST", "/api/scratch/start", map[string]string{"stone_id": stoneID})
	perCell := -1.0 // 第一次出現的正累積值 == 每格價值（裂紋先出現也只是 0×0.95）
	last := 0.0
	payout := 0.0
	for i := 0; i < 12; i++ {
		resp := c.do("POST", "/api/scratch/reveal", map[string]any{"stone_id": stoneID, "cell": i})
		acc := resp["accumulated"].(float64)
		if perCell < 0 && acc > 0 {
			perCell = acc
		}
		last = acc
		if resp["done"] == true {
			payout = resp["payout"].(float64)
		}
	}
	if perCell <= 0 {
		t.Fatalf("刮完 12 格累積值都是 0（每格價值 = %v）", perCell)
	}
	if last < 6*perCell {
		t.Fatalf("累積沒有累加（每格 %.0f、全開 %.0f）——刮石又只算一格的價值了", perCell, last)
	}
	if payout != last {
		t.Fatalf("done 的 payout(%v) 不等於累積值(%v)", payout, last)
	}
	t.Logf("每格 %.0f → 全開 %.0f（約 %.1f 格）", perCell, last, last/perCell)
}
