
import json, urllib.request, http.cookiejar, time, sqlite3, os
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
name="botchk_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
shop=call("GET","/api/shop"); it=shop["grades"][0]["items"][0]
call("POST","/api/shop/buy",{"stone_id":it["id"]})
before=call("GET","/api/me")["chips"]
call("POST","/api/market/list",{"stone_id":it["id"],"ask_price":1})
mkt=call("GET","/api/market")
mine=[l for l in mkt["listings"] if l["stone_id"]==it["id"]]
print("剛掛上還在市場:", bool(mine))
# 把掛單時間往回挪 6 分鐘（模擬「過了 5 分鐘沒人理」）
db=r"C:\jade-gamble\data\jade.db"
c=sqlite3.connect(db, timeout=20)
c.execute("UPDATE listings SET created_at = datetime('now','-6 minutes') WHERE stone_id=? AND sold=0", (it["id"],))
c.commit(); c.close()
mkt2=call("GET","/api/market")
still=[l for l in mkt2["listings"] if l["stone_id"]==it["id"]]
print("6 分鐘後還在市場:", bool(still), "→", "被 bot 收走了 ✅" if not still else "還沒被收 ❌")
buys=mkt2.get("bot_buys", [])
if buys: print("最近收料紀錄:", json.dumps(buys[0], ensure_ascii=False))
after=call("GET","/api/me")["chips"]
print(f"賣家籌碼 {before} → {after}（+{after-before}，ask=1 → 應為 0）")
# 後端 bot 名單
import re
names = re.findall(r'\{Key: "([a-z]+)", Name: "([^"]+)"', open(r"C:\jade-gamble\backend\domain\marketbot.go", encoding="utf-8").read())
print(f"bot 種類數: {len(names)} →", "、".join(n for _, n in names))
