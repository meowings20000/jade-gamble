import json, urllib.request, http.cookiejar, time

B = "http://127.0.0.1:3002"
cj = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = [("User-Agent", "Mozilla/5.0")]


def call(m, p, b=None):
    d = json.dumps(b).encode() if b is not None else None
    try:
        r = op.open(urllib.request.Request(B + p, data=d, method=m,
                                           headers={"Content-Type": "application/json"}), timeout=60)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code, "__body": e.read()[:150].decode("utf8", "ignore")}


call("POST", "/api/auth/mock", {"username": "gemcut%d" % int(time.time())})
before = call("GET", "/api/gems")
print("開始前：已收集 %s 種" % before.get("kinds_owned"))

gems = []
cuts = 0
for i in range(45):
    shelf = call("GET", "/api/shop")
    stones = [s for g in (shelf.get("grades") or []) for s in (g.get("items") or [])]
    if not stones:
        call("POST", "/api/shop/refresh")
        shelf = call("GET", "/api/shop")
        stones = [s for g in (shelf.get("grades") or []) for s in (g.get("items") or [])]
    if not stones:
        print("沒石頭可買了（第 %d 次）" % (i + 1))
        break
    st = min(stones, key=lambda s: s.get("price", 10**9))
    if cuts == 0 and i == 0:
        print("石頭欄位:", list(st.keys()))
    sid = st.get("id") or st.get("stone_id") or st.get("sid") or st.get("key")
    b = call("POST", "/api/shop/buy", {"id": sid})
    if b.get("__status"):
        print("買不到：", b)
        break
    r = call("POST", "/api/cut", {"id": sid})
    cuts += 1
    if r.get("gem"):
        gems.append((r.get("gem_name"), r.get("gem")))
        print("  💎 第 %d 刀切到 %s" % (cuts, r.get("gem_name")))

after = call("GET", "/api/gems")
print("切了 %d 刀 → 圖鑒：收集 %s/%s 種，共 %s 顆" % (cuts, after.get("kinds_owned"),
                                                  after.get("kinds_total"), after.get("count_total")))
for g in after.get("gems") or []:
    if g.get("count"):
        print("   ", g["name"], "×", g["count"], "首次", str(g.get("first_at"))[:10])
