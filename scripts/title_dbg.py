
import json, urllib.request, http.cookiejar, time
cj=http.cookiejar.CookieJar(); op=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj)); op.addheaders=[("User-Agent","Mozilla/5.0")]
def call(m,p,b=None):
    d=json.dumps(b).encode() if b is not None else None
    try:
        r=op.open(urllib.request.Request("http://127.0.0.1:3002"+p,data=d,method=m,headers={"Content-Type":"application/json"}),timeout=30)
        return r.status, r.read().decode()[:400]
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:400]
call("POST","/api/auth/mock",{"username":"tdbg_%d"%time.time()})
print("titles:", call("GET","/api/titles"))
print("history:", call("GET","/api/history")[0])
