# 賭石工坊 · Jade Gamble

純虛擬籌碼的賭石網頁遊戲（**不含任何真實金錢**）。一刀窮、一刀富——切、磨、刮三種開石賭法，競標場玩家對賭，異色圖鑑收集，雙榜排行。

**Play**: http://localhost:3002 （Docker 起在 127.0.0.1:3002）

## 玩法

| 玩法 | 賭法 | 賭感 |
|---|---|---|
| 切石 | 一刀全翻牌，x0 ~ x100 | 拉霸式一翻兩瞪眼 |
| 磨石 | crash 階梯 ×1.0→×8.0，磨崩全沒收，隨時落袋 | push-your-luck |
| 刮石 | 12 格逐格刮，裂紋跌價、可提前賣（-4%）、全開 1.08× | 資訊 vs 貪婪 |
| 套型 | 開完避裂取件（手鐲/平安扣/吊墜），避得好 ×1.6 | 益智收尾 |
| 競標場 | 玩家掛單轉賣，5% 手續費，價高者得 | PvP 識貨差價 |

## 經濟（防通膨設計）

- **莊家邊際藏在石價**：公斤料 EV 3.2%、表現料 6.7%、開窗料 9.4%（`domain/econ_test.go` 蒙地卡羅鎖定）
- **回收口**：遞增刷新費（300→600→1200→2400→4800 封頂）、打燈報告費（5% 石價）、掛單手續費 5%、兌換所（每項 EV < 售價）、套型加工費 5%
- **產出**：玩法賠率、每日登入 500、破產救濟（1,000 籌碼 或 刮到爽+300，每日 3 次）

## 系統

- **異色石**（獨立 roll，與品質相乘）：紫羅蘭 3% ×1.8 → 帝王綠 0.05% ×15；墨翠開前估值僅 1/3（賭性最強）
- **圖鑑收藏分**：首開計全額、再開 +10%；收藏家榜與財富榜雙榜
- **彩蛋**：卞和之石（公斤料 0.01% 強制帝王綠玻璃種）、B貨騙局（表現料 0.3% 完美皮殼切開全毀）、連三磚 →「賭徒之魂」、單日五漲 →「黃金瞳」、帝王綠進名人堂
- **兌換所**：刮到爽門票、保險券（磨崩退 50%）、雙倍券、免費刷新券、打燈大師卡、黃金瞳殘光、磨石手感、頭像框

## 架構

```
backend/   Go 1.25 + modernc.org/sqlite（純 Go，CGO_ENABLED=0）
  domain/  石頭生成、EV 表、磨石階梯、刮石、套型（12 個測試鎖定經濟）
  store/   SQLite（WAL, MaxOpenConns=1）、嚴格 JSON、會話 cookie
  api/     27 條 HTTP 路由；嚴格 JSON（decode 兩次防 trailing）
frontend/ 原生 JS + Canvas 程序化渲染（seed → 同顆石全端一致）
          零素材、零外部資源、零 build step；embed 進 Go binary
Dockerfile 單容器 golang:1.25-alpine → alpine:3.20（非 root）
```

- **防作弊**：石頭真值只在伺服器；客戶端只拿 `{seed, grade, hint, revealed[]}`，seed 只決定外觀，決定不了內容。API 測試斷言貨架/市場回應不得洩漏 `quality/variety/cracks`
- **競標一致性**：同 seed 全端畫出同一顆石（前端測試斷言 10 種異色調色板 + 渲染確定性）

## 驗證

```bash
cd backend && go test -race -count=1 ./...     # 19 tests
cd frontend && node frontend_test.js           # ids/外部資源/解析/27 條 API 路徑
cd .. && python scripts/e2e_smoke.py           # 對 live Docker 全流程（登入→買→切/刮/磨→掛單→得標→兌換→嚴格 JSON）
```

已知回歸守門：`TestShopRefreshNoDeadlock` —— MaxOpenConns(1) 下在 WithTx 閉包內呼叫非 Tx 方法 = 永久死鎖（v1 開發中真的發生過，修在 `store/fill.go` + `UsernameTx`）。

## Discord 登入

容器預設 `MOCK_AUTH=1`（訪客試玩）。接真 Discord：

1. 到 [Discord Developer Portal](https://discord.com/developers/applications) → 你的應用（Client ID `1549397605623275520`）→ **OAuth2** → 複製／重設 **Client Secret**。
2. 在同一頁 **Redirects** 加入：`https://jade.meowmeow12245ouo.dpdns.org/api/auth/discord/callback`（本機測試再加 `http://localhost:3002/api/auth/discord/callback`）。
3. 填入 `.env`：

```bash
DISCORD_CLIENT_ID=1549397605623275520
DISCORD_CLIENT_SECRET=貼上你的 Secret
DISCORD_REDIRECT_URI=http://localhost:3002/api/auth/discord/callback
```

4. `DISCORD_GUILD_ID` 是**公會白名單**：只有這個伺服器的成員能登入（設 `1544329028012474458` = 猪猪岛；留空則不限制）。
5. `docker compose -p jade-gamble up -d` 重啟，`/api/status` 的 `discord_oauth`／`discord_guild_restricted` 會變 `true`，登入頁的「用 Discord 登入」按鈕自動出現。

## 公網部署（Cloudflare Zero Trust）

固定網址：**https://jade.meowmeow12245ouo.dpdns.org**

- Tunnel：`jade-gamble`（id `1b797080-1a1b-4166-b6c4-d4b9d1fe7630`），**本機管理**（設定檔 `C:\Users\User\.cloudflared\jade-config.yml`）
- Ingress：`jade.meowmeow12245ouo.dpdns.org` → `http://localhost:3002`
- 開機自啟：啟動資料夾的 `jade-gamble-tunnel.vbs`（登入時隱藏啟動，不需管理員權限）
- 與 new-api 的 tunnel（`New API TUnnel`，dashboard 管理）完全獨立，互不影響

手動重啟通道：
```bash
"C:/Program Files (x86)/cloudflared/cloudflared.exe" --config "C:/Users/User/.cloudflared/jade-config.yml" tunnel run jade-gamble
```

## 部署筆記

本機 3000（new-api）、3001、8080、8081、8100 已被佔用，本專案用 **3002**。DB 在 `./data/jade.db`（volume）。
## 競標場規則

- **礦區直送（NPC 貨源）**：每位玩家擁有**獨立**的礦區池（固定 6 顆，買一顆補一顆），品質分佈與商店相同——真有好石，不是別人挑剩的。價格為商店標價的 1.25~1.6 倍。池子**不共通**：別人買了什麼、還剩什麼，你完全看不到，所以無法從「誰在搶哪顆」反推好石位置。
- **玩家掛單匿名**：競標場上看不到賣家身份（API 不回傳 seller 欄位）。賣家照樣收到 95% 貨款（5% 手續費），但買家永遠不知道是誰賣的——避免私下串通、也避免「這個人賣的一定是廢物」的聲譽效應。
- 傳統模式石頭（origin=classic）不可掛單。
- 每個玩家自己的池石只有自己能買；買別人的池子會直接擋掉。

## 磨石：力度必須配得上種水

種水在你買下石頭那一刻就定死了，所以「該怎麼磨」也跟著定死：

| 種水 | 吃得住的力度 |
|---|---|
| 玻璃種 / 冰種 | **重磨** |
| 油青種 / 豆種 | **正磨** |
| 磚頭料 | **輕磨** |

每條裂紋讓理想力度 −0.5，深裂再 −1（裂多的料只能輕手）。

- **開磨前選力度，選了不能改**。配得上，機器順暢一路上去；配不上，每層爆裂率 +18%。
- **手感會誠實告訴你配不配**（「機器順暢」vs「壓得太重了」vs「力度太輕，磨不動這料」），
  但不會直接說出種水——你得從打燈報告、皮殼表現和手感自己判斷。
- 每層爆裂率 = 種水基礎率（玻璃 14% → 磚頭 19%）+ 力度錯配 + 裂紋。
- 倍率 ×0.93 起、每層 ×1.20，**天花板由種水決定**（磚頭 ×5.0 → 玻璃種 ×8.0）。
- 經濟學（`TestPolishForceEconomics` 鎖定）：配對力度下玻璃種期望 ≈1.27、冰種 ≈1.13，
  磚頭料 ≈0.93；加權（讀對率 60~90%）EV 均 < 0.995，莊家仍有邊際。
