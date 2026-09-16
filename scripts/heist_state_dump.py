
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=60)
    return json.loads(r.read() or b"{}")
call("POST","/api/auth/mock",{"username":"dump%d"%int(time.time())})
call("POST","/api/heist/join",{"grade":0}); call("POST","/api/heist/fill")
call("POST","/api/heist/act",{"action":"betray"})
st=call("GET","/api/heist")
h=st.get("heist") or {}
print("heist keys:", sorted(h.keys()))
for k,v in h.items():
    if isinstance(v,list) and v and isinstance(v[0],dict):
        print("列表欄位:", k, "→ 每項 keys:", sorted(v[0].keys()))
        for it in v: print("   ", json.dumps(it,ensure_ascii=False))
print("me:", json.dumps(st.get("me") or {},ensure_ascii=False))
