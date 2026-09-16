
import json, urllib.request, http.cookiejar, time
B = "http://127.0.0.1:3002"
def mk(name):
    cj = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
    op.addheaders = [("User-Agent", "Mozilla/5.0")]
    def call(m, p, b=None):
        d = json.dumps(b).encode() if b is not None else None
        try:
            r = op.open(urllib.request.Request(B + p, data=d, method=m,
                                               headers={"Content-Type": "application/json"}), timeout=60)
            return json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            return {"__status": e.code}
    call("POST", "/api/auth/mock", {"username": name})
    return call
ts = int(time.time())
A, Bc, C, D = [mk(f"hz{i}_{ts}") for i in range(4)]
print("入場費/獎池:", json.dumps(A("GET", "/api/heist")["fees"], ensure_ascii=False), A("GET","/api/heist")["pots"])
for c, n in [(A,"A"),(Bc,"B"),(C,"C"),(D,"D")]:
    r = c("POST", "/api/heist/join", {"grade": 1})
    print(f"{n} 入場:", r.get("message"), "| 籌碼", r.get("chips"))
# 第一輪：全部合作
for c in (A, Bc, C, D):
    c("POST", "/api/heist/act", {"action": "cooperate"})
st = A("GET", "/api/heist")
h = st["heist"]
print(f"第一輪後 進度 {h['progress']}/{h['target']} | 活著 {sum(1 for o in st['others'] if o['alive']) + (1 if st['me']['alive'] else 0)} 人")
# 第二輪：A 背叛 B（B 警戒）
A("POST", "/api/heist/act", {"action": "betray", "target": st["others"][0]["user_id"]})
st2 = A("GET", "/api/heist")
btarget = st2["others"][0]["user_id"]
print("A 選了背叛對象 user_id:", btarget)
