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


call("POST", "/api/auth/mock", {"username": "cv%d" % int(time.time())})
for attempt in range(8):
    shop = call("GET", "/api/shop")
    items = [(it, gi) for gi, g in enumerate(shop.get("grades", [])) for it in g.get("items", [])]
    if not items:
        print("沒貨"); break
    it, gi = min(items, key=lambda x: x[0]["price"])
    call("POST", "/api/shop/buy", {"stone_id": it["id"]})
    st = call("POST", "/api/polish/start", {"stone_id": it["id"], "force": 2})
    if "error" in st:
        print("開磨失敗:", st["error"]); continue
    print("買入價 %d｜BaseValue %s｜開磨顯示落袋會拿到 %s" % (it["price"], st.get("base_value"), st.get("cash_value")))
    ok, alive = True, True
    for _ in range(3):
        adv = call("POST", "/api/polish/advance", {"stone_id": it["id"]})
        if "error" in adv or not adv.get("alive"):
            print("  磨崩（救回 %s）" % adv.get("salvage")); ok = False; break
        print("  第 %s 層：倍率 %.3f｜畫面會顯示落袋實拿 %s｜成本 %s"
              % (adv.get("stage"), adv.get("multiplier", 0), adv.get("cash_value"), adv.get("stone_price")))
    if not ok:
        continue
    before = call("GET", "/api/me").get("chips")
    cash = call("POST", "/api/polish/cash", {"stone_id": it["id"]})
    after = call("GET", "/api/me").get("chips")
    print("  === 落袋：畫面預告 %s｜實際入帳 %s｜錢包 %d → %d（+%d）"
          % (adv.get("cash_value"), cash.get("payout"), before, after, after - before))
    print("  一致嗎:", "✅ 一致" if int(adv.get("cash_value") or -1) == int(cash.get("payout") or -2) == (after - before) else "❌ 不一致")
    break
