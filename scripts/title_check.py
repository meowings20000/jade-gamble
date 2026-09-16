
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
name="title_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
d=call("GET","/api/titles")
print("稱號總數:", len(d["titles"]), "| 目前裝備:", repr(d["equipped"]))
print("新帳號解鎖:", [t["name"] for t in d["titles"] if t["unlocked"]])
# 切幾刀看解鎖變化 + 裝備
shop=call("GET","/api/shop"); it=shop["grades"][0]["items"][0]
call("POST","/api/shop/buy",{"stone_id":it["id"]})
call("POST","/api/cut",{"stone_id":it["id"]})
d2=call("GET","/api/titles")
print("切一刀後解鎖:", [t["name"] for t in d2["titles"] if t["unlocked"]])
print("裝上:", call("POST","/api/titles/equip",{"key":"first_cut"})["message"])
print("me.title:", call("GET","/api/me")["title"])
print("沒解鎖的不能裝:", call("POST","/api/titles/equip",{"key":"tycoon"}).get("error"))
print("卸下:", call("POST","/api/titles/equip",{"key":""})["message"])
