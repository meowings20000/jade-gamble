
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=40)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__err": e.read()[:200].decode("utf8","none")}
call("POST","/api/auth/mock",{"username":"res%d"%int(time.time())})
shop=call("GET","/api/shop"); it=None
for g in shop.get("grades",[]):
    for x in g.get("items",[]):
        if it is None or x["price"]<it["price"]: it=x
call("POST","/api/shop/buy",{"stone_id":it["id"]})
s1=call("POST","/api/polish/start",{"stone_id":it["id"],"force":1})
print("開磨:", {k:s1.get(k) for k in ("stage","multiplier","force_name")})
for _ in range(2):
    a=call("POST","/api/polish/advance",{"stone_id":it["id"]})
    if not a.get("alive"): print("磨崩了"); break
    print("  推進 → stage", a.get("stage"), "×", a.get("multiplier"))
s2=call("POST","/api/polish/start",{"stone_id":it["id"]})   # 不帶 force＝續磨
print("回來再按磨石:", {k:s2.get(k) for k in ("stage","multiplier","force_name","error")})
