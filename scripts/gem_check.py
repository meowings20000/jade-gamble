
import json, urllib.request, http.cookiejar, time, collections
UA={"User-Agent":"Mozilla/5.0"}
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=list(UA.items())
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try: return {"__status": e.code, **json.loads(e.read() or b"{}")}
        except Exception: return {"__status": e.code}
name="gem_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
gems=[]; n=0
for i in range(90):
    me=call("GET","/api/me")
    if me.get("chips",0) < 1500:
        call("POST","/api/relief",{"option":"chips"})
        if call("GET","/api/me").get("chips",0) < 1500: break
    shop=call("GET","/api/shop")
    items=shop["grades"][0]["items"]
    if not items:
        # 貨架買空是設計（不再跳頁免費補）：付費刷新繼續
        call("POST","/api/shop/refresh",{"grade":0})
        shop=call("GET","/api/shop")
        items=shop["grades"][0]["items"]
        if not items: break
    it=items[0]
    call("POST","/api/shop/buy",{"stone_id":it["id"]})
    res=call("POST","/api/cut",{"stone_id":it["id"]})
    if "payout" not in res: continue
    n+=1
    if res.get("gem"):
        gems.append((res["gem"], res["gem_name"], res["payout"], it["price"], res.get("egg")))
print(f"切了 {n} 顆，彩蛋 {len(gems)} 次 = {len(gems)/n*100:.1f}%（期望 5%）")
for g in gems: print("  💎", g)
