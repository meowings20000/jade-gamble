
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=90)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__err": e.read()[:200].decode("utf8","ignore")}
call("POST","/api/auth/mock",{"username":"chat%d"%int(time.time())})
a=call("POST","/api/bank/apply",{"amount":30000,"hours":24,"reason":"我要買石頭"})
print("申請:", a.get("decision"), "|", str(a.get("message"))[:60])
r=call("POST","/api/bank/appeal",{"message":"利率可以低一點嗎喵"})
print("申訴回應:", json.dumps(r, ensure_ascii=False)[:240])
st=call("GET","/api/bank")
print("對話筆數:", len(st.get("chat") or []))
for m in (st.get("chat") or [])[:6]:
    print("   ", m.get("role"), ":", str(m.get("content"))[:60])
