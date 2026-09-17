
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=90)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code, "__body": e.read()[:400].decode("utf8","ignore")}
call("POST","/api/auth/mock",{"username":"say%d"%int(time.time())})
call("POST","/api/bank/apply",{"amount":30000,"hours":24,"reason":"我想借來買石頭"})
print("say 前 chat:", json.dumps(call("GET","/api/bank").get("chat"), ensure_ascii=False)[:120])
r = call("POST","/api/bank/say",{"text":"利率可以低一點嗎喵"})
print("say 回應:", json.dumps(r, ensure_ascii=False)[:500])
after = call("GET","/api/bank")
print("say 後 chat 長度:", len(after.get("chat") or []), "| 內容:", json.dumps(after.get("chat"), ensure_ascii=False)[:300])
