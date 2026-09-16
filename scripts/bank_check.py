
import json, urllib.request, http.cookiejar, time
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
call("POST","/api/auth/mock",{"username":"bank_%d"%time.time()})
d=call("GET","/api/bank")
print("條款:", json.dumps(d["terms"], ensure_ascii=False))
print("AI 已接:", d["terms"]["ai"])
r=call("POST","/api/bank/apply",{"amount":20000,"hours":24,"reason":"我要去賭一塊好料，三天內翻本"})
print("申請結果:", r.get("decision"), "|", r.get("message"))
print("籌碼:", call("GET","/api/me")["chips"], "| 到期要還:", r.get("repay_total"))
l=r.get("loan")
if l:
    a=call("POST","/api/bank/appeal",{"message":"老闆娘行行好，利率算低一點喵"})
    print("申訴1:", a.get("decision"), "|", a.get("message"), "| 剩", a.get("appeals_left"), "輪")
    a2=call("POST","/api/bank/appeal",{"message":"我一定會還的，拜託拜託喵"})
    print("申訴2:", a2.get("decision"), "| 剩", a2.get("appeals_left"), "輪")
    print("還款:", call("POST","/api/bank/repay",{}).get("message"))
    print("還完後:", call("GET","/api/bank")["loan"])
    print("壞輸入(借太多):", call("POST","/api/bank/apply",{"amount":999999,"hours":99,"reason":"x"}).get("error"))
