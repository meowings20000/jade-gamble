
import json, urllib.request, http.cookiejar, time
UA={"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
B="https://jade.meowmeow12245ouo.dpdns.org"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=list(UA.items())
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try: return {"__status": e.code, **json.loads(e.read() or b"{}")}
        except Exception: return {"__status": e.code}
call("POST","/api/auth/mock",{"username":"yb_%d"%time.time()})
o=call("GET","/api/yboss/odds")
print("Y佬賠率:")
for e in o.get("cut",[]): print(f"  {e['label']:<6} ×{e['mult']:<5} {round(e['prob']*100)}%")
print("市場欄位:", end=" ")
mkt=call("GET","/api/market")
print(sorted(mkt["listings"][0].keys()) if mkt.get("listings") else "無掛單", "| npc 外洩:", any("npc" in l for l in mkt.get("listings",[])))
# Y佬切一刀大樣本勝率
wins=0; n=0
for _ in range(30):
    r=call("POST","/api/yboss/bet",{"stake":100,"choice":"cut"})
    if "mult" in r:
        n+=1
        if r["mult"]>=1.0: wins+=1
print(f"Y佬切一刀實測: {wins}/{n} = {wins/n:.0%} 勝" if n else "沒骰到")
# 紀錄端點
h=call("GET","/api/history")
print("紀錄:", "entries" in h, "stats:", h.get("stats"))
