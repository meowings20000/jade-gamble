package main

// 勝率模擬器：用「跟正式服務同一份」的 domain 程式碼直接跑，
// 不靠真人玩、不受喵喵幣限制，因此可以跑到上萬顆把誤差壓到 ±0.5%。
// 用法：cd backend && go run ./cmd/sim [每檔顆數]

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"

	"jade-gamble/backend/domain"
)

// polishReport: 磨石每層期望值（配對／差一級），含新舊規則對照。
// 舊規則＝崩了整顆報廢（報酬 0）；新規則＝救回當前倍率 35%。
func polishReport() {
	qs := []domain.Quality{domain.Brick, domain.Bean, domain.OilGreen, domain.Icy, domain.Glass}
	fmt.Println("\n== 磨石每層期望值（EV/step）==")
	fmt.Println("種水\t基礎爆裂\t配對EV(舊)\t配對EV(新)\t差一級EV(新)\t天花板倍率")
	for _, q := range qs {
		st := &domain.Stone{Quality: q}
		p := domain.PolishBaseBreakFor(st)
		pw := p + 0.18
		if pw > 0.95 {
			pw = 0.95
		}
		oldMatched := 1.20 * (1 - p)
		newMatched := (1-p)*1.25 + p*0.35
		newWrong := (1-pw)*1.25 + pw*0.35
		fmt.Printf("%s\t%.1f%%\t%.4f\t%.4f\t%.4f\t%.2f×\n",
			q.Name(), p*100, oldMatched, newMatched, newWrong, domain.PolishCeilingFor(st))
	}
	fmt.Println("（配對 EV > 1 ＝長期值得磨；差一級應 < 1 ＝選錯力度該立刻收手）")
}

func main() {
	n := 20000
	if len(os.Args) > 1 {
		if v, err := strconv.Atoi(os.Args[1]); err == nil && v > 0 {
			n = v
		}
	}
	labels := map[domain.ShopGrade]string{
		domain.KiloGrade:    "公斤料（蒙頭／入門）",
		domain.FeatureGrade: "表現料（中階）",
		domain.WindowGrade:  "開窗料（高階）",
	}
	rnd := rand.New(rand.NewSource(20260917))
	fmt.Printf("== 切石勝率模擬（每檔 %d 顆，同一份 domain 程式碼）==\n", n)
	tw, tn, tp, tpr := 0, 0, 0, 0
	for _, g := range []domain.ShopGrade{domain.KiloGrade, domain.FeatureGrade, domain.WindowGrade} {
		wins, cnt, payout, price := 0, 0, 0, 0
		mults := make([]float64, 0, n)
		brick := 0
		for i := 0; i < n; i++ {
			st := domain.GenerateStone(g, rnd)
			p := domain.CutReveal(st, false)
			if st.Price <= 0 {
				continue
			}
			cnt++
			payout += p
			price += st.Price
			m := float64(p) / float64(st.Price)
			mults = append(mults, m)
			if p >= st.Price {
				wins++
			}
			if st.Quality == domain.Brick {
				brick++
			}
		}
		sort.Float64s(mults)
		p := float64(wins) / float64(cnt)
		se := math.Sqrt(p * (1 - p) / float64(cnt))
		fmt.Printf("%s\n  勝率 %.1f%%（%d/%d，±%.2f%% 95%%CI）| EV %.4f\n",
			labels[g], p*100, wins, cnt, 1.96*se*100, float64(payout)/float64(price))
		fmt.Printf("  倍率：中位 %.2f｜P10 %.2f｜P90 %.2f｜最好 %.2f｜最差 %.2f｜磚頭料 %.1f%%\n",
			mults[len(mults)/2], mults[len(mults)/10], mults[len(mults)*9/10], mults[len(mults)-1], mults[0],
			float64(brick)/float64(cnt)*100)
		tw += wins
		tn += cnt
		tp += payout
		tpr += price
	}
	fmt.Printf("全部：勝率 %.1f%%（%d/%d）| EV %.4f\n",
		float64(tw)/float64(tn)*100, tw, tn, float64(tp)/float64(tpr))
	polishReport()
}
