
import json, urllib.request, http.cookiejar, time
B = "http://127.0.0.1:3002"
cj = http.cookiejar.CookieJar(); op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = [("User-Agent", "Mozilla/5.0")]
def call(m, p, b=None):
    d = json.dumps(b).encode() if b is not None else None
    try:
        r = op.open(urllib.request.Request(B + p, data=d, method=m, headers={"Content-Type": "application/json"}), timeout=60)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code, "__body": e.read()[:200].decode("utf8", "ignore")}
call("POST", "/api/auth/mock", {"username": "e2e%d" % int(time.time())})
print("入場:", call("POST", "/api/heist/join", {"grade": 0}).get("message"))
print("按「直接開局」:", call("POST", "/api/heist/fill").get("message"))
st = call("GET", "/api/heist"); h = st.get("heist") or {}
print("開局後: 狀態", h.get("status"), "| 位子", h.get("seats"), "| 進度", h.get("progress"), "/", h.get("target"))
for rnd in range(6):
    if (st.get("heist") or {}).get("status") == "done":
        break
    call("POST", "/api/heist/act", {"action": "cooperate"})
    st = call("GET", "/api/heist"); h = st.get("heist") or {}
    me = st.get("me") or {}
    print("  第%d次出手 → 狀態 %s | 進度 %s/%s | 我活著 %s | 賠付 %s" % (rnd + 1, h.get("status"), h.get("progress"), h.get("target"), me.get("alive"), me.get("payout")))
print("=== 結束後（畫面應該還留在桌上，不再彈回大廳）===")
st = call("GET", "/api/heist"); h = st.get("heist") or {}
print("還看得到桌子:", bool(h), "| 狀態:", h.get("status"), "| 我的賠付:", (st.get("me") or {}).get("payout"), "| 籌碼(喵喵幣):", call("GET", "/api/me").get("chips"))
