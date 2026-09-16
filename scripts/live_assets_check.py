
import urllib.request, re
UA={"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126 Safari/537.36"}
B="https://jade.meowmeow12245ouo.dpdns.org"
def get(p, hdrs=None):
    r = urllib.request.urlopen(urllib.request.Request(B+p, headers={**UA, **(hdrs or {})}), timeout=25)
    return r.status, dict(r.headers), r.read().decode("utf-8", "replace")
s,h,html = get("/")
print("index:", s)
print("  轉賬按鈕:", 'data-view="transfer"' in html, "| 控制臺:", 'nav-admin' in html, "| 活動橫幅:", 'event-banner' in html)
print("  cache headers:", {k:v for k,v in h.items() if k.lower() in ("cache-control","etag","last-modified")})
s,_,js = get("/app.js")
print("app.js:", s, "loadTransfers:", "loadTransfers" in js, "| loadAdmin:", "loadAdmin" in js, "| esc:", "function esc(" in js)
print("  app.js cache:", {k:v for k,v in _ .items()} if False else "")
