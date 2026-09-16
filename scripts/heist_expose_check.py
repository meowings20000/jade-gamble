
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
            return {"__status": e.code, "__body": e.read()[:120].decode("utf8","ignore")}
    call("POST","/api/auth/mock",{"username":n}); return call
ts=int(time.time()//1)
A=mk("A%d"%ts); Bc=mk("B%d"%ts)
print("A 入場:", A("POST","/api/heist/join",{"grade":0}).get("message"))
print("B 入場:", Bc("POST","/api/heist/join",{"grade":0}).get("message"))
A("POST","/api/heist/fill")
st=A("GET","/api/heist"); h=st.get("heist") or {}
others=[o for o in (st.get("others") or []) if isinstance(o,dict)]
print("同桌其他人:", [o.get("name") for o in others][:4])
bid=[o["user_id"] for o in others if "B%d"%ts in str(o.get("name"))]
print("B 的座位:", bid[:1])
print("B 投背叛打 A:", Bc("POST","/api/heist/act",{"action":"betray","target":bid[0] if bid else 0}).get("message"))
print("A 出手合作:", A("POST","/api/heist/act",{"action":"cooperate"}).get("message"))
st=A("GET","/api/heist"); h=st.get("heist") or {}
me=st.get("me") or {}
print("A 看到 tried_to_kill_me:", me.get("tried_to_kill_me"), "| hunters 名字:", st.get("hunters"), "| 進度:", h.get("progress"), "/", h.get("target"))
