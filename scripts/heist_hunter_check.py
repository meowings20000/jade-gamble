
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
def mk(n):
    cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
    def call(m,p,b=None):
        d=json.dumps(b).encode() if b is not None else None
        try:
            r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=60)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            return {"err": e.code}
    call("POST","/api/auth/mock",{"username":n}); return call
ts=int(time.time()); nA="Killer%d"%ts; nB="Victim%d"%ts
A=mk(nA); Bc=mk(nB)
print("A 入場:", A("POST","/api/heist/join",{"grade":0}).get("message"))
print("B 入場:", Bc("POST","/api/heist/join",{"grade":0}).get("message"))
A("POST","/api/heist/fill")
stA=A("GET","/api/heist"); oA=[o for o in (stA.get("others") or []) if isinstance(o,dict)]
bid=[o["user_id"] for o in oA if nB in str(o.get("name"))]
print("A 看到的 B 座位:", bid[:1])
r=A("POST","/api/heist/act",{"action":"betray","target":(bid[0] if bid else 0)})
print("A 投背叛打 B:", r.get("message") or r)
print("B 出手合作:", Bc("POST","/api/heist/act",{"action":"cooperate"}).get("message"))
st=Bc("GET","/api/heist"); me=st.get("me") or {}
print("B 看到 tried_to_kill_me:", me.get("tried_to_kill_me"), "| hunters 名字:", st.get("hunters"), "| 應該等於", nA)
