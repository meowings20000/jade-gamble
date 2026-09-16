
import json, urllib.request, http.cookiejar, time
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
name="wait_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
shop=call("GET","/api/shop"); it=shop["grades"][0]["items"][0]
call("POST","/api/shop/buy",{"stone_id":it["id"]})
call("POST","/api/market/list",{"stone_id":it["id"],"ask_price":1})
print("掛單:", it["id"], flush=True)
for i in range(8):
    time.sleep(60)
    mkt=call("GET","/api/market")
    here=any(l["stone_id"]==it["id"] for l in mkt["listings"])
    print(f"{i+1} 分鐘：還在市場={here}", flush=True)
    if not here:
        print("✅ bot 收走了（5 分鐘門檻生效）")
        print("收料紀錄:", json.dumps([b for b in mkt.get("bot_buys",[]) if b["stone_id"]==it["id"]], ensure_ascii=False))
        break
