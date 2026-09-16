
import json, urllib.request, http.cookiejar, time
B = "http://127.0.0.1:3002"
def mk(n):
    cj = http.cookiejar.CookieJar(); op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
    def call(m, p, b=None):
        d = json.dumps(b).encode() if b is not None else None
        try:
            r = op.open(urllib.request.Request(B+p, data=d, method=m, headers={"Content-Type":"application/json"}), timeout=60)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            return {"__status": e.code}
    call("POST","/api/auth/mock",{"username":n}); return call
for g in [0, 1]:
    P = mk("pay%d_%d" % (g, int(time.time())))
    P("POST","/api/heist/join",{"grade":g}); P("POST","/api/heist/fill")
    for _ in range(8):
        st = P("GET","/api/heist"); h = st.get("heist") or {}
        if h.get("status") == "done": break
        P("POST","/api/heist/act",{"action":"cooperate"})
    st = P("GET","/api/heist"); h = st.get("heist") or {}; me = st.get("me") or {}
    lists = {k: v for k, v in h.items() if isinstance(v, list)}
    seatlist = None
    for v in lists.values():
        if v and isinstance(v[0], dict) and "alive" in v[0]:
            seatlist = v; break
    alive = sum(1 for s in (seatlist or []) if s.get("alive"))
    pot, tgt, prog = h.get("pot") or 0, h.get("target") or 0, h.get("progress") or 0
    print("檔位%d｜入場 %s｜進度 %s/%s｜狀態 %s｜活著 %s｜我分到 %s" % (g, h.get("entry"), prog, tgt, h.get("status"), alive, me.get("payout")))
    if me.get("payout") and tgt and prog < tgt and alive:
        eff = pot // max(1, alive)
        print("   折算檢查：獎池 %d ÷ 活著 %d × (%d/%d) = %d  ← 我實拿 %d" % (pot, alive, prog, tgt, int(round(eff * prog / tgt)), me.get("payout")))
    elif me.get("payout") and tgt and prog >= tgt:
        print("   挖到滿進度 (%d/%d) → 拿滿：獎池 %d ÷ 活著 %d = %d" % (prog, tgt, pot, alive, pot // max(1, alive)))
