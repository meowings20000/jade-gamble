
import json, urllib.request, http.cookiejar, time, sqlite3
UA={"User-Agent":"Mozilla/5.0"}
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=list(UA.items())
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request("http://127.0.0.1:3002"+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        try: return {"__status": e.code, **json.loads(e.read() or b"{}")}
        except Exception: return {"__status": e.code}
name="seize_%d"%time.time()
call("POST","/api/auth/mock",{"username":name})
# 買幾顆石頭當財產
for i in range(4):
    s=call("GET","/api/shop")
    if not s["grades"][0]["items"]: call("POST","/api/shop/refresh",{"grade":0}); s=call("GET","/api/shop")
    it=s["grades"][0]["items"][0]
    call("POST","/api/shop/buy",{"stone_id":it["id"]})
before=call("GET","/api/me")
stones_before=len(before["items"])
r=call("POST","/api/bank/apply",{"amount":20000,"hours":24,"reason":"測試逾期"})
loan=r.get("loan") or {}
chips_after_loan=call("GET","/api/me")["chips"]
print(f"借款前籌碼 {before['chips']} → 借款後 {chips_after_loan}（+20000）")
print(f"石頭 {stones_before} 顆 → 倉庫 {stones_before} 顆")
# 把到期時間挪到過去
c=sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=20)
c.execute("UPDATE loans SET due_at = datetime('now','-1 hours') WHERE id=?", (loan.get("id"),))
c.commit()
print("到期時間已改成 1 小時前")
c.close()
d=call("GET","/api/bank")   # 這一步會執行沒收
me=call("GET","/api/me")
owned=len([s for s in me["items"].values() if s.get("state")=="owned"]) if isinstance(me["items"], dict) else len(me["items"])
print(f"沒收後籌碼 {me['chips']}（應 ≈ {chips_after_loan//2}，一半）")
print(f"沒收後倉庫石頭 {owned} 顆（原本 {stones_before}，應剩一半）")
print("貸款狀態:", (call("GET","/api/bank")["history"] or [{}])[0].get("status"))
