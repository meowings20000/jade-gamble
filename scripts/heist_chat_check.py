
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
def mk(n):
    cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
    def c(m,p,b=None):
        d=json.dumps(b).encode() if b is not None else None
        try:
            r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            return {"__err": e.read()[:160].decode("utf8","replace")}
    c("POST","/api/auth/mock",{"username":n}); return c
ts=int(time.time())
a = mk("chatA%d"%ts); b = mk("chatB%d"%ts)
a("POST","/api/heist/join",{"grade":0}); b("POST","/api/heist/join",{"grade":0})
a("POST","/api/heist/fill",{"grade":0})
print("A 講話:", json.dumps(a("POST","/api/heist/say",{"text":"我先合作，別殺我喵"}), ensure_ascii=False)[:150])
print("太快再講:", json.dumps(a("POST","/api/heist/say",{"text":"喂"}).get("__err","(竟然成功)"), ensure_ascii=False)[:80])
time.sleep(2.2)
print("B 講話:", json.dumps(b("POST","/api/heist/say",{"text":"我信你一次"}), ensure_ascii=False)[:80])
st = b("GET","/api/heist")
for m in (st.get("chat") or []):
    print("   ", m.get("name"), ":", m.get("text"))
