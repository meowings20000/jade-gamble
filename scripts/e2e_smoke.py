#!/usr/bin/env python3
"""E2E smoke against the live Docker stack on 127.0.0.1:3002."""
import json
import urllib.request
import http.cookiejar

BASE = 'http://127.0.0.1:3002'
cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))


def call(method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header('Content-Type', 'application/json')
    data = json.dumps(body).encode() if body is not None else None
    with opener.open(req, data) as r:
        return json.loads(r.read().decode())


# 1. login (mock)
import time as _t
r = call('POST', '/api/auth/mock', {'username': 'e2e_smoke_' + str(_t.time_ns())})
assert r['ok'], r
print('✓ login:', r['username'])

# 2. me
me = call('GET', '/api/me')
print('✓ signup chips:', me['chips'])
assert me['chips'] == 10000

# 3. shop
shop = call('GET', '/api/shop')
g0 = shop['grades'][0]
print('✓ kilo shelf:', [i['id'] for i in g0['items']])
item = g0['items'][0]
assert 'quality' not in item, 'shelf leaks truth!'

# 4. buy + cut
r = call('POST', '/api/shop/buy', {'stone_id': item['id']})
print('✓ bought', item['id'], 'chips now', r['chips'])
cut = call('POST', '/api/cut', {'stone_id': item['id']})
print('✓ cut:', cut['quality'], cut['variety'], 'payout', cut['payout'], 'mult', cut['multiplier'])

# 5. buy second + scratch all 12 cells
shop = call('GET', '/api/shop')
item2 = shop['grades'][0]['items'][0]
call('POST', '/api/shop/buy', {'stone_id': item2['id']})
st = call('POST', '/api/scratch/start', {'stone_id': item2['id']})
for cell in range(12):
    rr = call('POST', '/api/scratch/reveal', {'stone_id': item2['id'], 'cell': cell})
    if rr.get('done') and rr.get('payout') is not None:
        print('✓ scratch done: quality', rr['quality'], 'payout', rr['payout'])
        break
    assert rr['accumulated'] >= 0

# 6. buy third + polish loop
shop = call('GET', '/api/shop')
item3 = shop['grades'][0]['items'][0]
call('POST', '/api/shop/buy', {'stone_id': item3['id']})
call('POST', '/api/polish/start', {'stone_id': item3['id'], 'force': 2})
for i in range(10):
    rr = call('POST', '/api/polish/advance', {'stone_id': item3['id']})
    if not rr['alive']:
        print('✓ polish broke at stage', rr['broke_at'], '(refund', rr.get('insurance_refund', 0), ')')
        break
else:
    rr = call('POST', '/api/polish/cash', {'stone_id': item3['id']})
    print('✓ polished to top, cashed', rr['payout'])

# 7. market roundtrip with second user
cj2 = http.cookiejar.CookieJar()
op2 = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj2))
def call2(method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header('Content-Type', 'application/json')
    data = json.dumps(body).encode() if body is not None else None
    with op2.open(req, data) as r:
        return json.loads(r.read().decode())
call2('POST', '/api/auth/mock', {'username': 'e2e_buyer_' + str(_t.time_ns())})
shop = call('GET', '/api/shop')
item4 = shop['grades'][0]['items'][0]
call('POST', '/api/shop/buy', {'stone_id': item4['id']})
call('POST', '/api/market/list', {'stone_id': item4['id'], 'ask_price': 500})
mkt = call2('GET', '/api/market')
l = next((x for x in mkt['listings'] if x['stone_id'] == item4['id']), None)
assert l is not None, 'seller listing not visible'
assert 'seller' not in l and 'seller_id' not in l, 'listing must be anonymous'
print('✓ listing up (匿名):', l['ask_price'])
# 礦區直送 pool is private per player
npc_b = [x for x in mkt['listings'] if x.get('npc')]
assert npc_b, 'buyer sees no 礦區直送 stock'
r = call2('POST', '/api/market/buy', {'listing_id': l['id']})
print('✓ buyer got', r['stone_id'])

# 8. exchange + leaderboard + collection + hall
ex = call('GET', '/api/exchange')
print('✓ exchange catalog:', len(ex['catalog']), 'items')
r = call('POST', '/api/exchange/buy', {'key': 'frenzy_ticket'})
print('✓ frenzy ticket: 10 stones →', r['total'])
lb = call('GET', '/api/leaderboard?kind=wealth')
print('✓ leaderboard entries:', len(lb['entries']))
col = call('GET', '/api/collection')
print('✓ collection score:', col['score'], 'discovered:', sum(1 for v in col['varieties'] if v['discovered']))
hall = call('GET', '/api/hall')
print('✓ hall of fame entries:', len(hall['entries']))

# 9. relief path: drain chips by repeated refreshes until broke
me = call('GET', '/api/me')
chips = me['chips']
guard = 0
while chips > 0 and guard < 25:
    shop = call('GET', '/api/shop')
    g2 = shop['grades'][2]
    try:
        rr = call('POST', '/api/shop/refresh', {'grade': 2})
    except Exception:
        break
    # buy window stones to drain
    for it in g2['items']:
        try:
            r = call('POST', '/api/shop/buy', {'stone_id': it['id']})
            chips = r['chips']
        except urllib.error.HTTPError:
            pass
    me = call('GET', '/api/me')
    chips = me['chips']
    guard += 1
if chips == 0:
    r = call('POST', '/api/relief', {'option': 'chips'})
    print('✓ relief granted:', r)
    me = call('GET', '/api/me')
    assert me['chips'] == 1000
else:
    print('- relief path skipped (still has chips:', chips, ')')

# 10. strict JSON rejected
req = urllib.request.Request(BASE + '/api/auth/mock', method='POST',
                             data=b'{"username":"a"}{"username":"b"}')
req.add_header('Content-Type', 'application/json')
try:
    opener.open(req)
    raise SystemExit('✗ trailing JSON accepted!')
except urllib.error.HTTPError as e:
    assert e.code == 400
    print('✓ strict JSON rejected')

print('\nALL E2E CHECKS PASSED')