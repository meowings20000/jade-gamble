import json, urllib.request, http.cookiejar, time

B = "http://127.0.0.1:3002"
cj = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = [("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36")]


def call(m, p, b=None):
    d = json.dumps(b).encode() if b is not None else None
    try:
        r = op.open(urllib.request.Request(B + p, data=d, method=m,
                                           headers={"Content-Type": "application/json"}), timeout=40)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try:
            return json.loads(e.read() or b"{}")
        except Exception:
            return {"__err": str(e)}


def chips():
    return call("GET", "/api/me").get("chips")


call("POST", "/api/auth/mock", {"username": "ledger%d" % int(time.time())})
print("起始喵喵幣:", chips())

for attempt in range(1, 6):
    shop = call("GET", "/api/shop")
    cheapest = None
    for gi, g in enumerate(shop.get("grades", [])):
        for it in g.get("items", []):
            if cheapest is None or it["price"] < cheapest[0]["price"]:
                cheapest = (it, gi)
    if not cheapest:
        print("商店沒貨")
        break
    it, gi = cheapest
    price = it["price"]
    b0 = chips()
    call("POST", "/api/shop/buy", {"stone_id": it["id"]})
    b1 = chips()
    print("\n--- 第 %d 次嘗試（檔位 %d）---" % (attempt, gi))
    print("買入價 %d → 喵喵幣 %d → %d（扣了 %d）" % (price, b0, b1, b0 - b1))

    st = call("POST", "/api/polish/start", {"stone_id": it["id"], "force": 2})
    if "error" in st:
        print("  開磨失敗:", st["error"])
        continue
    print("  開磨倍率:", st.get("start"), "| 每層成長:", st.get("gain"), "| 爆裂率:", round(st.get("break_prob", 0) * 100, 1), "%")
    ok = True
    for _ in range(3):
        adv = call("POST", "/api/polish/advance", {"stone_id": it["id"]})
        if "error" in adv:
            print("  推進失敗:", adv["error"]); ok = False; break
        print("  第 %d 層：倍率 %.3f｜喵喵幣 %d" % (adv.get("stage", 0), adv.get("multiplier", 0), chips()))
        if not adv.get("alive"):
            print("  磨崩了！救回:", adv.get("salvage"), "保險:", adv.get("insurance_refund"), "| 喵喵幣", chips())
            ok = False
            break
    if not ok:
        continue
    print("  落袋前喵喵幣:", chips())
    cash = call("POST", "/api/polish/cash", {"stone_id": it["id"]})
    print("  落袋回應:", json.dumps({k: v for k, v in cash.items() if k in ("payout", "chips", "error", "multiplier")}, ensure_ascii=False))
    print("  落袋後喵喵幣:", chips(), "→ 這一輪淨變化: %+d" % (chips() - b0))
    break
