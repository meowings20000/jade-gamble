
import json, urllib.request, http.cookiejar, time
UA={"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
B="https://jade.meowmeow12245ouo.dpdns.org"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=list(UA.items())
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return r.status, json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read() or b"{}")
call("POST","/api/auth/mock",{"username":"hint_%d"%time.time()})
_,shop=call("GET","/api/shop")
for g in shop["grades"]:
    it=g["items"][0]
    print(f"grade {g['grade']} 免費描述 = {it['hint']!r}")
_,lk=call("POST","/api/shop/light",{"stone_id":shop["grades"][0]["items"][0]["id"]})
print("公斤料打燈(付費後) =", repr(lk.get("report")), "cost=", lk.get("cost"))
_,shop2=call("GET","/api/shop")
print("打燈後卡片 hint =", repr(shop2["grades"][0]["items"][0]["hint"]))
_,lw=call("POST","/api/shop/light",{"stone_id":shop["grades"][2]["items"][0]["id"]})
print("開窗料打燈 =", repr(lw.get("report")))
print("開窗料免費描述 =", repr(shop["grades"][2]["items"][0]["hint"]))
_,mk=call("GET","/api/market")
print("市場掛單描述 =", repr(mk["listings"][0]["light_hint"]) if mk["listings"] else "(無掛單)")
