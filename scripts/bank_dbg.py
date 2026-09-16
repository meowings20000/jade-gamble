
import json, urllib.request, http.cookiejar, time
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request("http://127.0.0.1:3002"+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return r.status, r.read().decode()[:300]
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:300]
call("POST","/api/auth/mock",{"username":"bk2_%d"%time.time()})
print("apply:", call("POST","/api/bank/apply",{"amount":20000,"hours":24,"reason":"我要去賭一塊好料"}))
