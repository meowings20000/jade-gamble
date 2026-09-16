
import json, urllib.request, http.cookiejar, time, sqlite3
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
name="dbg_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
shop=call("GET","/api/shop"); it=shop["grades"][0]["items"][0]
call("POST","/api/shop/buy",{"stone_id":it["id"]})
call("POST","/api/market/list",{"stone_id":it["id"],"ask_price":1})
print("stone:", it["id"], "price:", it["price"])
db=r"C:\jade-gamble\data\jade.db"
c=sqlite3.connect(db, timeout=20)
c.execute("UPDATE listings SET created_at=datetime('now','-10 minutes') WHERE stone_id=? AND sold=0",(it["id"],))
c.commit()
row=list(c.execute("SELECT id, seller_id, ask_price, sold, created_at FROM listings WHERE stone_id=?",(it["id"],)))
print("DB 掛單:", row)
print("DB stone state:", list(c.execute("SELECT state, owner_id, quality, price FROM stones WHERE id=?",(it["id"],))))
c.close()
call("GET","/api/market")
c=sqlite3.connect(db, timeout=20)
print("呼叫市場後 sold:", list(c.execute("SELECT id, sold FROM listings WHERE stone_id=?",(it["id"],))))
print("bot_buys 有沒有這顆:", list(c.execute("SELECT bot_name, price, note, created_at FROM bot_buys WHERE stone_id=?",(it["id"],))))
c.close()
mkt=call("GET","/api/market")
print("市場還看得到:", any(l["stone_id"]==it["id"] for l in mkt["listings"]))
