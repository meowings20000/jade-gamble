
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
def mk(name):
    cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
    def call(m,p,b=None):
        d=json.dumps(b).encode() if b is not None else None
        try:
            r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=60)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            return {"__err": e.read()[:160].decode("utf8","ignore")}
    call("POST","/api/auth/mock",{"username":name}); return call
ts=int(time.time())
nameB="rlSink%d"%ts
mk(nameB)                      # 先把收款的帳號建出來
a = mk("rlA%d"%ts)
for i in range(1,5):
    chips = int(a("GET","/api/me").get("chips",0))
    if chips >= 5000:
        tr = a("POST","/api/transfer",{"to":nameB,"amount":chips-3000})
        print("  轉出後:", a("GET","/api/me").get("chips"), "| 轉賬回應:", json.dumps(tr, ensure_ascii=False)[:70])
    r = a("POST","/api/relief",{"option":"chips"})
    print("第%d次領救濟: %s | 喵喵幣=%s" % (i, json.dumps(r, ensure_ascii=False)[:110], a("GET","/api/me").get("chips")))
