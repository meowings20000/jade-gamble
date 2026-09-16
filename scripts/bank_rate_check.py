
import json, urllib.request, http.cookiejar, time
cj = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = [("User-Agent", "Mozilla/5.0")]
def call(m, p, b=None):
    d = json.dumps(b).encode() if b is not None else None
    try:
        r = op.open(urllib.request.Request("http://127.0.0.1:3002" + p, data=d, method=m,
                                           headers={"Content-Type": "application/json"}), timeout=60)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code}
call("POST", "/api/auth/mock", {"username": "rlad_%d" % time.time()})
call("POST", "/api/bank/apply", {"amount": 20000, "hours": 24, "rate": 0.30, "reason": "我要進貨一批料"})
o = call("GET", "/api/bank").get("offer") or {}
r0 = o.get("rate") or 0
print("申請提議: 利率 %.0f%% | 還 %d" % (r0 * 100, (o.get("principal") or 0) + (o.get("interest") or 0)))
for i, msg in enumerate(["我常來的算便宜點喵", "保證準時還降一點喵", "快沒錢了幫幫我喵"]):
    call("POST", "/api/bank/appeal", {"message": msg})
    o = call("GET", "/api/bank").get("offer") or {}
    r = o.get("rate") or 0
    print("申訴%d 後: 利率 %.0f%% | 還 %d | 已用 %s 輪" % (i + 1, r * 100,
          (o.get("principal") or 0) + (o.get("interest") or 0), o.get("appeals")))
