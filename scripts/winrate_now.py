import json, urllib.request, http.cookiejar, time, statistics, math

B = "http://127.0.0.1:3002"
UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}


def mk(name):
    cj = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
    op.addheaders = list(UA.items())

    def call(m, p, b=None):
        d = json.dumps(b).encode() if b is not None else None
        try:
            r = op.open(urllib.request.Request(B + p, data=d, method=m,
                                               headers={"Content-Type": "application/json"}), timeout=30)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            try:
                return json.loads(e.read() or b"{}")
            except Exception:
                return {}
    call("POST", "/api/auth/mock", {"username": name})
    return call


def cuts(call, grade_idx, want):
    """切 want 顆，回傳 (wins, n, payout_sum, price_sum, mults)"""
    wins = n = psum = prsum = 0
    mults = []
    tries = 0
    while n < want and tries < want * 6:
        tries += 1
        shop = call("GET", "/api/shop")
        try:
            g = shop["grades"][grade_idx]
        except Exception:
            break
        if not g.get("items"):
            call("POST", "/api/shop/refresh", {"grade": grade_idx})
            continue
        it = g["items"][0]
        me = call("GET", "/api/me")
        if me.get("chips", 0) < it["price"]:
            rc = call("POST", "/api/relief", {"option": "chips"})
            if "chips_given" not in rc:
                break  # 沒錢也沒救濟了
            continue
        if "error" in call("POST", "/api/shop/buy", {"stone_id": it["id"]}):
            call("POST", "/api/shop/refresh", {"grade": grade_idx})
            continue
        res = call("POST", "/api/cut", {"stone_id": it["id"]})
        if "payout" not in res:
            continue
        n += 1
        psum += res["payout"]
        prsum += it["price"]
        mults.append(res["payout"] / it["price"])
        if res["payout"] >= it["price"]:
            wins += 1
    return wins, n, psum, prsum, mults


LAB = {0: "公斤料(蒙頭/入門)", 1: "表現料(中階)", 2: "開窗料(高階)"}
TARGET = {0: 14, 1: 12, 2: 8}
totals = {0: [0, 0, 0, 0, []], 1: [0, 0, 0, 0, []], 2: [0, 0, 0, 0, []]}
accts = 5
for i in range(accts):
    call = mk("wr%d_%d" % (int(time.time()), i))
    for gid in (0, 1, 2):
        w, n, ps, prs, ms = cuts(call, gid, TARGET[gid])
        t = totals[gid]
        t[0] += w; t[1] += n; t[2] += ps; t[3] += prs; t[4] += ms
    print("  帳號 %d/%d 完成" % (i + 1, accts), flush=True)

print("\n=== 主玩法切石 勝率（即時對局實測）===")
for gid in (0, 1, 2):
    w, n, ps, prs, ms = totals[gid]
    if not n:
        print("%s: 沒樣本" % LAB[gid]); continue
    p = w / n
    se = math.sqrt(p * (1 - p) / n)
    print("%s: 勝 %d/%d = %.1f%% (±%.1f%% 95%%CI) | EV=%.3f | 中位倍率 %.2f | 最好 %.2f 最差 %.2f"
          % (LAB[gid], w, n, p * 100, 1.96 * se * 100, ps / prs, statistics.median(ms), max(ms), min(ms)))
allw = sum(totals[g][0] for g in totals); alln = sum(totals[g][1] for g in totals)
allp = sum(totals[g][2] for g in totals); allpr = sum(totals[g][3] for g in totals)
if alln:
    print("整體: 勝 %d/%d = %.1f%% | EV=%.3f（樣本 %d）" % (allw, alln, allw / alln * 100, allp / allpr, alln))
