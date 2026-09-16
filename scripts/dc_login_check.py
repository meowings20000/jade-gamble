
import json, urllib.request, urllib.parse, re
UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126 Safari/537.36"}
BASE = "https://jade.meowmeow12245ouo.dpdns.org"

def get(url, hdrs=None, data=None, method="GET"):
    req = urllib.request.Request(url, data=data, method=method, headers={**UA, **(hdrs or {})})
    try:
        r = urllib.request.urlopen(req, timeout=25)
        return r.status, dict(r.headers), r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), e.read().decode("utf-8", "replace")

# 1) 狀態
s, _, b = get(BASE + "/api/status")
print("status:", s, b.strip())

# 2) authorize 轉址參數
s, h, _ = get(BASE + "/api/auth/discord")
loc = h.get("Location", "")
print("authorize HTTP:", s, "→", (loc.split("?")[0] if loc else "(no redirect)"))
q = urllib.parse.parse_qs(urllib.parse.urlparse(loc).query)
for k in ("client_id", "redirect_uri", "scope", "response_type"):
    print(f"  {k} = {q.get(k, [''])[0]}")
print("  state 存在:", bool(q.get("state")))

# 3) 憑證 + redirect_uri 登記狀態（實際打 Discord token 端點）
env = dict(l.strip().split("=", 1) for l in open(r"C:\jade-gamble\.env", encoding="utf-8") if "=" in l and not l.strip().startswith("#"))
body = urllib.parse.urlencode({
    "client_id": env["DISCORD_CLIENT_ID"],
    "client_secret": env["DISCORD_CLIENT_SECRET"],
    "grant_type": "authorization_code",
    "code": "dummy_code_probe",
    "redirect_uri": env["DISCORD_REDIRECT_URI"],
}).encode()
s, _, b = get("https://discord.com/api/oauth2/token", {"Content-Type": "application/x-www-form-urlencoded"}, body, "POST")
print("token probe HTTP:", s)
try:
    j = json.loads(b)
    err = j.get("error"); desc = j.get("error_description")
    print("  error:", err)
    print("  description:", desc)
    if err == "invalid_grant":
        print("  >>> redirect_uri 已登記、憑證正確（只差沒給真的 code）")
    elif desc and "redirect_uri" in str(desc):
        print("  >>> redirect_uri 沒登記在 Discord Portal")
    elif err == "invalid_client":
        print("  >>> client_id/secret 不配對")
except Exception as e:
    print("  parse fail:", b[:200])
