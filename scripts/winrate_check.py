
import json, urllib.request, http.cookiejar, time, statistics
UA={"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
B="https://jade.meowmeow12245ouo.dpdns.org"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=list(UA.items())
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try: return json.loads(e.read() or b"{}")
        except Exception: return {}
name="wr_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
call("POST","/api/transfer",{"to":"meowings2000","amount":0})  # 忽略
def buy_and_cut(grade_idx, rounds):
    wins=0; n=0; payout_sum=0; price_sum=0; mults=[]
    for _ in range(rounds):
        shop=call("GET","/api/shop")
        g=shop["grades"][grade_idx]
        if not g["items"]: 
            call("POST","/api/shop/refresh",{"grade":grade_idx}); continue
        it=g["items"][0]
        me=call("GET","/api/me")
        if me.get("chips",0) < it["price"]*3:
            # 沒錢就用救濟/換便宜檔
            call("POST","/api/relief",{"option":"chips"})
            me=call("GET","/api/me")
            if me.get("chips",0) < it["price"]: break
        call("POST","/api/shop/buy",{"stone_id":it["id"]})
        res=call("POST","/api/cut",{"stone_id":it["id"]})
        if "payout" not in res: continue
        n+=1; payout_sum+=res["payout"]; price_sum+=it["price"]
        mults.append(res["payout"]/it["price"])
        if res["payout"]>=it["price"]: wins+=1
    return wins,n,payout_sum,price_sum,mults
for idx,label in [(0,"公斤料"),(1,"表現料")]:
    w,n,p,pr,mults=buy_and_cut(idx, 26)
    if n:
        print(f"{label}: {w}/{n} 勝 = {w/n:.0%} | EV = {p/pr:.3f} | 中位倍率 {statistics.median(mults):.2f} | 最差 {min(mults):.2f} 最好 {max(mults):.2f}")
