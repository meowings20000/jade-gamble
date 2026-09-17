
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=90)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code, "__body": e.read()[:300].decode("utf8","ignore")}
call("POST","/api/auth/mock",{"username":"bank%d"%int(time.time())})
print("1) 錢莊狀態:", json.dumps(call("GET","/api/bank"), ensure_ascii=False)[:300])
print("2) 申請 20000 / 3 小時:")
r = call("POST","/api/bank/apply",{"amount":20000,"hours":3,"reason":"我想周轉一下買石頭"})
print("   回應:", json.dumps(r, ensure_ascii=False)[:700])
