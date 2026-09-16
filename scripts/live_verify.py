"""Live 驗收：對跑著的服務（預設公網）逐項實測，不用肉眼。

    python scripts/live_verify.py [base_url]

檢查項目：
  1. /api/status 與 Discord 設定
  2. 開局籌碼 5000..50000 區間（新帳號應為 50000）
  3. 刮石必須累加（每格→全開的成長倍率）
  4. 轉賬：無條件即時、附條件需接受、拒絕退還
  5. 拍賣 bot 收料紀錄欄位
  6. 活動公告 / 管理員權限
  7. 三檔料的勝率與期望值（切石）
  8. 開窗料打燈要有具體提示（種水/裂紋），公斤料不用
"""
import json
import random
import sys
import time
import urllib.error
import urllib.request
import http.cookiejar

UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
BASE = sys.argv[1] if len(sys.argv) > 1 else "https://jade.meowmeow12245ouo.dpdns.org"

PASS, FAIL = [], []


def check(name, ok, detail=""):
    (PASS if ok else FAIL).append(name)
    print(f"{'PASS' if ok else 'FAIL'}  {name}" + (f"  — {detail}" if detail else ""))


class Client:
    def __init__(self, name):
        self.name = name
        self.jar = http.cookiejar.CookieJar()
        self.op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
        self.op.addheaders = list(UA.items())
        self.call("POST", "/api/auth/mock", {"username": name})

    def call(self, method, path, body=None, expect_fail=False):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(BASE + path, data=data, method=method,
                                     headers={"Content-Type": "application/json"})
        try:
            r = self.op.open(req, timeout=30)
            return r.status, json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            raw = e.read()
            try:
                return e.code, json.loads(raw or b"{}")
            except Exception:
                return e.code, {"raw": raw[:200].decode("utf-8", "replace")}


stamp = str(int(time.time()))
print(f"=== live verify @ {BASE} ===\n")

# 1) status
_, st = Client.__new__(Client).__class__ and (0, {}) or (0, {})
c0 = Client("lv_%s" % stamp)
_, status = c0.call("GET", "/api/status")
check("Discord OAuth 開啟", status.get("discord_oauth") is True, str(status))
check("Discord 公會白名單開啟", status.get("discord_guild_restricted") is True)

# 2) 開局籌碼
_, me = c0.call("GET", "/api/me")
start_chips = me.get("chips", 0)
check("新帳號籌碼 = 50000", start_chips == 50000, f"chips={start_chips}")
check("/api/me 有 is_admin 欄位", "is_admin" in me, str(list(me.keys())[:8]))

# 3) 刮石累加
_, shop = c0.call("GET", "/api/shop")
grades = shop["grades"]
kilo = grades[0]
item = kilo["items"][0]
c0.call("POST", "/api/shop/buy", {"stone_id": item["id"]})
_, s = c0.call("POST", "/api/scratch/start", {"stone_id": item["id"]})
per_cell, last, payout = None, 0, 0
for i in range(12):
    _, resp = c0.call("POST", "/api/scratch/reveal", {"stone_id": item["id"], "cell": i})
    acc = resp.get("accumulated", 0)
    if per_cell is None and acc > 0:
        per_cell = acc
    last = acc
    if resp.get("done"):
        payout = resp.get("payout", 0)
check("刮石有累加（≥6 格價值）", bool(per_cell) and last >= 6 * per_cell,
      f"每格 {per_cell} → 全開 {last}（{last / per_cell if per_cell else 0:.1f} 格）payout={payout}")
check("刮石 payout 與累積一致", payout == last, f"payout={payout} acc={last}")

# 8) 開窗料 vs 公斤料的打燈具體度
hint_kilo = s.get("hint", "")
_, shop2 = c0.call("GET", "/api/shop")
win_item = None
for g in shop2["grades"]:
    if g.get("grade") == 2 and g["items"]:
        win_item = g["items"][0]
if win_item is None and len(shop2["grades"]) > 2 and shop2["grades"][2]["items"]:
    win_item = shop2["grades"][2]["items"][0]
if win_item:
    c0.call("POST", "/api/shop/buy", {"stone_id": win_item["id"]})
    _, rep = c0.call("POST", "/api/shop/light", {"stone_id": win_item["id"]})
    win_hint = rep.get("hint") or rep.get("report") or json.dumps(rep, ensure_ascii=False)
    explicit = ("【開窗" in win_hint) or ("種" in win_hint and ("裂" in win_hint or "紋" in win_hint))
    check("開窗料打燈有具體提示（種水＋裂紋）", explicit, win_hint[:80])
else:
    check("開窗料打燈有具體提示（種水＋裂紋）", False, "找不到開窗料")

# 4) 轉賬
peer = Client("lv_peer_%s" % stamp)
_, a0 = c0.call("GET", "/api/me")
_, b0 = peer.call("GET", "/api/me")
# 先把錢湊夠（買石頭把錢花掉了）
if a0["chips"] < 8000:
    need = 12000 - a0["chips"]
    peer.call("POST", "/api/transfer", {"to": c0.name, "amount": need})
    _, a0 = c0.call("GET", "/api/me")
c0.call("POST", "/api/transfer", {"to": peer.name, "amount": 3000})
_, a1 = c0.call("GET", "/api/me")
_, b1 = peer.call("GET", "/api/me")
check("無條件轉賬即時到賬", a1["chips"] == a0["chips"] - 3000 and b1["chips"] == b0["chips"] + 3000,
      f"{a0['chips']}→{a1['chips']} / {b0['chips']}→{b1['chips']}")
r2 = c0.call("POST", "/api/transfer", {"to": peer.name, "amount": 2000, "condition": "幫我刮三格"})[1]
_, a2 = c0.call("GET", "/api/me")
_, inc = peer.call("GET", "/api/transfers")
if not inc.get("incoming"):
    check("附條件轉賬先託管、對方待接受", False, f"沒有待接受的轉賬: {r2}")
    pending = None
else:
    pending = inc["incoming"][0]
if pending:
    check("附條件轉賬先託管、對方待接受",
          a2["chips"] == a1["chips"] - 2000 and pending["condition"] == "幫我刮三格",
          f"condition={pending['condition']!r}")
    peer.call("POST", "/api/transfer/decline", {"id": pending["id"]})
_, a3 = c0.call("GET", "/api/me")
check("拒絕後全額退還", a3["chips"] == a2["chips"] + 2000, f"{a2['chips']}→{a3['chips']}")

# 5) 拍賣場 bot 欄位
_, mkt = c0.call("GET", "/api/market")
check("競標場回傳 bot_buys 欄位", "bot_buys" in mkt, f"keys={list(mkt.keys())}")
check("競標場掛單不含賣家（匿名）",
      all("seller" not in l for l in mkt["listings"]), f"{len(mkt['listings'])} 張單")

# 6) 活動公告 / 管理權限
_, ev = c0.call("GET", "/api/events")
check("活動公告端點可用（一般玩家）", isinstance(ev.get("events"), list))
code, _ = c0.call("GET", "/api/admin/panel")
check("非管理員進不了控制臺", code >= 300, f"HTTP {code}")

# 7) 三檔勝率（切石）
def winrate(grade_idx, rounds=14):
    wins = 0
    n = 0
    for _ in range(rounds):
        _, sh = c0.call("GET", "/api/shop")
        if grade_idx >= len(sh["grades"]) or not sh["grades"][grade_idx]["items"]:
            continue
        it = sh["grades"][grade_idx]["items"][0]
        price = it["price"]
        if c0.call("GET", "/api/me")[1]["chips"] < price * 2:
            break
        c0.call("POST", "/api/shop/buy", {"stone_id": it["id"]})
        _, res = c0.call("POST", "/api/cut", {"stone_id": it["id"]})
        n += 1
        if res.get("payout", 0) >= price:
            wins += 1
    return wins, n

for idx, label in [(0, "公斤料"), (1, "表現料"), (2, "開窗料")]:
    w, n = winrate(idx)
    if n:
        rate = w / n
        check(f"{label}勝率接近 6:4（0.5~0.72）", 0.5 <= rate <= 0.72, f"{w}/{n} = {rate:.0%}")

print(f"\n=== {len(PASS)} passed, {len(FAIL)} failed ===")
if FAIL:
    print("FAILED:", ", ".join(FAIL))
    sys.exit(1)
