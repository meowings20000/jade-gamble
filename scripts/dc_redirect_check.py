
import urllib.request, urllib.parse
UA = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
class NoRedir(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None
op = urllib.request.build_opener(NoRedir, urllib.request.HTTPSHandler())
op.addheaders = list(UA.items())
try:
    r = op.open("https://jade.meowmeow12245ouo.dpdns.org/api/auth/discord", timeout=25)
    print("HTTP", r.status, "(沒轉址)")
    print(r.read().decode()[:200])
except urllib.error.HTTPError as e:
    print("HTTP", e.code, "Location:", e.headers.get("Location", "")[:220])
