
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
call("POST","/api/auth/mock",{"username":"ck_%d"%time.time()})
mkt=call("GET","/api/market")
print("掛單欄位:", sorted(mkt["listings"][0].keys()) if mkt.get("listings") else "無")
print("有 npc 欄位嗎:", any("npc" in l for l in mkt.get("listings", [])))
print("有 seller 欄位嗎:", any("seller" in l for l in mkt.get("listings", [])))
h=call("GET","/api/history")
print("紀錄端點:", "entries" in h, "| stats:", h.get("stats"))
# 切一顆看紀錄會不會長出來
shop=call("GET","/api/shop"); it=shop["grades"][0]["items"][0]
call("POST","/api/shop/buy",{"stone_id":it["id"]})
res=call("POST","/api/cut",{"stone_id":it["id"]})
h2=call("GET","/api/history")
print("切完後紀錄筆數:", h2["stats"]["total"], "| 最新一筆:", h2["entries"][0] if h2.get("entries") else None)
