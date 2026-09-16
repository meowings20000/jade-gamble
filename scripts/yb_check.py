
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
call("POST","/api/auth/mock",{"username":"yb2_%d"%time.time()})
print("odds raw:", json.dumps(call("GET","/api/yboss/odds"), ensure_ascii=False)[:400])
wins=0;n=0;mults=[]
for _ in range(60):
    r=call("POST","/api/yboss/bet",{"stake":100,"choice":"cut"})
    if "mult" in r:
        n+=1; mults.append(r["mult"])
        if r["mult"]>=1.0: wins+=1
import collections
print(f"切一刀 {wins}/{n} = {wins/n:.0%} 勝 | 倍率分布 {dict(collections.Counter(mults))}")
