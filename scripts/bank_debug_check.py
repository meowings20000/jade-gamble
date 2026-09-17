
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=90)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__err": e.read()[:250].decode("utf8","ignore")}
call("POST","/api/auth/mock",{"username":"dbg%d"%int(time.time())})
a=call("POST","/api/bank/apply",{"amount":30000,"hours":24,"reason":"買石頭"})
print("申請:", a.get("decision"))
s=call("POST","/api/bank/appeal",{"message":"利率低一點喵"})
print("申訴:", s.get("decision"), "|", str(s.get("message"))[:40])
d=call("GET","/api/bank/_debug")
print("loans:", json.dumps(d.get("loans"), ensure_ascii=False)[:400])
print("loan_chat 最後幾列:", json.dumps(d.get("recent_chat"), ensure_ascii=False)[:400])
print("/api/bank 讀到的 chat 筆數:", len(call("GET","/api/bank").get("chat") or []))
