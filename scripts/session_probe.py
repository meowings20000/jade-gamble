
import json, urllib.request, http.cookiejar
UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
B = "https://jade.meowmeow12245ouo.dpdns.org"

cj = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
op.addheaders = list(UA.items())

body = json.dumps({"username": "sess_probe_live"}).encode()
req = urllib.request.Request(B + "/api/auth/mock", data=body, method="POST",
                             headers={"Content-Type": "application/json"})
r = op.open(req, timeout=25)
print("mock login:", r.status)
for c in cj:
    print(f"  cookie: name={c.name} domain={c.domain} path={c.path} secure={c.secure} "
          f"expires={c.expires} rest={c._rest}")
r2 = op.open(B + "/api/me", timeout=25)
d = json.loads(r2.read())
print("me:", r2.status, "user:", d.get("username"), "chips:", d.get("chips"), "is_admin:", d.get("is_admin"))
