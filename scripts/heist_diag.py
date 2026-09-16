
import json, urllib.request, http.cookiejar, sqlite3, time
cj = http.cookiejar.CookieJar(); op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = [("User-Agent", "Mozilla/5.0")]
def call(m, p, b=None):
    d = json.dumps(b).encode() if b is not None else None
    try:
        r = op.open(urllib.request.Request("http://127.0.0.1:3002" + p, data=d, method=m,
                    headers={"Content-Type": "application/json"}), timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__status": e.code, "__body": e.read()[:300].decode("utf8", "ignore")}
print("mock 登入:", call("POST", "/api/auth/mock", {"username": f"dg{int(time.time())}"}).get("username"))
me = call("GET", "/api/me")
print("me:", json.dumps({k: me.get(k) for k in ("id", "username", "chips")}, ensure_ascii=False))
print("join:", json.dumps(call("POST", "/api/heist/join", {"grade": 0}), ensure_ascii=False))
print("state:", json.dumps(call("GET", "/api/heist"), ensure_ascii=False)[:400])
c = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=30)
uid = me.get("id")
print("DB 這個人的 seat:", c.execute("SELECT heist_id,user_id,alive,action,entry FROM heist_seats WHERE user_id=?", (uid,)).fetchall())
print("DB 所有 heists:", c.execute("SELECT id,grade,status,pot,progress FROM heists ORDER BY id DESC LIMIT 4").fetchall())
print("DB seat 總數:", c.execute("SELECT heist_id, COUNT(*) FROM heist_seats GROUP BY heist_id").fetchall())
print("模擬 MyHeist SQL:", c.execute("""SELECT h.id, h.status FROM heists h JOIN heist_seats s ON s.heist_id=h.id
  WHERE s.user_id=? AND h.status IN ('open','running')
  AND (SELECT COUNT(*) FROM heist_seats s2 WHERE s2.heist_id=h.id) <= 4
  ORDER BY h.id DESC LIMIT 1""", (uid,)).fetchall())
c.close()
