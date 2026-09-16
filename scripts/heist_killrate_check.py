
import json, urllib.request, http.cookiejar, time, collections
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
kills = misses = 0
for i in range(60):
    ts = "%d_%d" % (int(time.time()*1000) % 100000, i)
    A = mk("KA" + ts); V = mk("VA" + ts)
    A("POST","/api/heist/join",{"grade":0}); V("POST","/api/heist/join",{"grade":0})
    stA = A("GET","/api/heist"); oA = [o for o in (stA.get("others") or []) if isinstance(o,dict)]
    vid = [o["user_id"] for o in oA if str(o.get("name")).startswith("VA")]
    if not vid: continue
    A("POST","/api/heist/fill")
    A("POST","/api/heist/act",{"action":"betray","target":vid[0]})
    V("POST","/api/heist/act",{"action":"cooperate"})
    stA = A("GET","/api/heist")
    me = stA.get("me") or {}
    h = stA.get("heist") or {}
    # 看被攻擊者 V 是否還活著
    stV = V("GET","/api/heist")
    vme = stV.get("me") or {}
    if vme.get("alive") is False: kills += 1
    elif vme.get("alive") is True: misses += 1
    if (i+1) % 20 == 0:
        tot = kills + misses
        print("前 %d 次：死亡 %d / 失手 %d → 成功率 %.1f%%（目標 70%%）" % (tot, kills, misses, 100.0*kills/max(1,tot)))
