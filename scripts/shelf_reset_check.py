
import json, urllib.request, http.cookiejar, time
B="http://127.0.0.1:3002"
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def c(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request(B+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        return {"__err": e.read()[:200].decode("utf8","replace")}
c("POST","/api/auth/mock",{"username":"shelf%d"%int(time.time())})
def nextprice():
    s=c("GET","/api/shop"); return s["grades"][0]["next_refresh"], s["chips"]
# 情境A：先看商店（貨架有石頭）→ 刷新會漲價
p0,ch0 = nextprice()
for i in range(3):
    st, ch = c("POST","/api/shop/refresh",{"grade":0}), None
p1,ch1 = nextprice()
print("A 有石頭：刷新 3 次後，下一刷價格 %s → %s（喵喵幣 %s → %s）" % (p0, p1, ch0, ch1))
# 情境B：新帳號直接刷新（貨架還沒補貨＝賣光狀態）→ 應免費且價格歸零
c("POST","/api/auth/mock",{"username":"shelfB%d"%int(time.time())})
b0 = c("GET","/api/me")["chips"]
r = c("POST","/api/shop/refresh",{"grade":0})
b1 = c("GET","/api/me")["chips"]
s = c("GET","/api/shop")
print("B 賣光狀態：刷新回應 %s｜喵喵幣 %s → %s（扣 %d）｜下一刷價格 %s" %
      (json.dumps(r)[:60], b0, b1, b0-b1, s["grades"][0]["next_refresh"]))
