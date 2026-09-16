
import urllib.request
UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
class NoRedir(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *a, **k): return None
op = urllib.request.build_opener(NoRedir, urllib.request.HTTPSHandler())
op.addheaders = list(UA.items())
B = "https://jade.meowmeow12245ouo.dpdns.org"
for path, label in [
    ("/api/auth/discord/callback?code=bogus&state=bogus", "callback 壞 code"),
    ("/api/auth/discord/callback?error=access_denied&error_description=user+denied", "callback 使用者按拒絕"),
    ("/api/auth/discord/callback", "callback 完全沒參數"),
]:
    try:
        r = op.open(B + path, timeout=25)
        print(f"{label}: HTTP {r.status} body={r.read().decode()[:120]}")
    except urllib.error.HTTPError as e:
        print(f"{label}: HTTP {e.code} Location={e.headers.get('Location','')[:90]}")
