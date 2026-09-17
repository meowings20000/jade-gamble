
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
call("POST","/api/auth/mock",{"username":"bankacc%d"%int(time.time())})
a = call("POST","/api/bank/apply",{"amount":20000,"hours":3,"reason":"我要買石頭"})
print("申請結果:", a.get("decision"), "|", a.get("message"))
oid = (a.get("offer") or {}).get("id")
print("按「接受」:", json.dumps(call("POST","/api/bank/accept",{"id":oid}), ensure_ascii=False)[:220])
me = call("GET","/api/bank")
print("接受後喵喵幣:", me.get("chips"), "| 貸款:", json.dumps(me.get("loan"), ensure_ascii=False)[:160])
print("拒絕測試:", json.dumps(call("POST","/api/bank/reject",{"id":oid}), ensure_ascii=False)[:120])
