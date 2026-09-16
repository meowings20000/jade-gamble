
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
            return {"__status": e.code, "__body": e.read()[:120].decode("utf8","ignore")}
    call("POST", "/api/auth/mock", {"username": n}); return call
ts = int(time.time()); P = mk("lv%d" % ts)
a = P("GET","/api/me"); print("初始喵喵幣:", a.get("chips"))
print("1) 入場:", P("POST","/api/heist/join",{"grade":0}).get("message"))
print("2) 直接開局:", P("POST","/api/heist/fill").get("message"))
st = P("GET","/api/heist"); h = st.get("heist") or {}
print("   行動倒數欄位 round_left =", h.get("round_left"), "秒 | action_sec =", h.get("action_sec"))
for i in range(6):
    if (P("GET","/api/heist").get("heist") or {}).get("status") == "done": break
    P("POST","/api/heist/act",{"action":"cooperate"})
st = P("GET","/api/heist"); h = st.get("heist") or {}
print("3) 玩完 → 狀態:", h.get("status"), "| 我分到:", (st.get("me") or {}).get("payout"))
print("4) 按「離開桌子」（結束的桌也該能離開）:", P("POST","/api/heist/leave").get("message"))
st = P("GET","/api/heist"); print("   離開後還看得到桌嗎:", bool(st.get("heist")), "| 大廳可再入場:", "heist" not in st or st.get("heist") is None)
after = P("GET","/api/me"); print("5) 結算後喵喵幣:", after.get("chips"))
print("   我的奪寶紀錄:", P("GET","/api/heist")["history"][:1] if P("GET","/api/heist").get("history") else "（空）")
