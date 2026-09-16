
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
def mk(n):
    cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
    def call(m,p,b=None):
        d=json.dumps(b).encode() if b is not None else None
        try:
            r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=60)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e: return {"err": e.code}
    call("POST","/api/auth/mock",{"username":n}); return call
kill=counter=nothing=0
for i in range(90):
    ts="%d_%d"%(int(time.time()*1000)%100000, i)
    A=mk("KA"+ts); V=mk("VA"+ts)
    A("POST","/api/heist/join",{"grade":0}); V("POST","/api/heist/join",{"grade":0})
    stA=A("GET","/api/heist"); oA=[o for o in (stA.get("others") or []) if isinstance(o,dict)]
    vid=[o["user_id"] for o in oA if str(o.get("name")).startswith("VA")]
    if not vid: continue
    A("POST","/api/heist/fill")
    A("POST","/api/heist/act",{"action":"betray","target":vid[0]})
    V("POST","/api/heist/act",{"action":"cooperate"})
    av=(A("GET","/api/heist").get("me") or {}).get("alive")
    vv=(V("GET","/api/heist").get("me") or {}).get("alive")
    if av is False and vv is True: counter+=1      # 攻擊者死＝被反殺
    elif vv is False and av is True: kill+=1       # 目標死＝刺殺成功
    elif av is True and vv is True: nothing+=1     # 都沒死＝無事發生
    tot=kill+counter+nothing
    if tot in (30,60,90):
        print("前 %d 次：成功 %d(%.0f%%) / 被反殺 %d(%.0f%%) / 無事 %d(%.0f%%)  ← 目標 60/20/20" % (
            tot, kill, 100.0*kill/tot, counter, 100.0*counter/tot, nothing, 100.0*nothing/tot))
