#!/usr/bin/env python3
"""Live check of the new 競標場 rules against the deployed stack:
   1. 礦區直送 pool is per-player (no shared stone IDs)
   2. player listings are anonymous (no seller field)
   3. a player can't buy another player's pool stone
   4. buying a pool stone works and the pool refills
"""
import json
import time
import urllib.request
import http.cookiejar

UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36'
BASE = 'http://127.0.0.1:3002'


def mk():
    return urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def call(op, method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header('Content-Type', 'application/json')
    req.add_header('User-Agent', UA)
    req.add_header('User-Agent', UA)
    data = json.dumps(body).encode() if body is not None else None
    try:
        with op.open(req, data) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode())


ts = str(time.time_ns())[-9:]
A, B = mk(), mk()
call(A, 'POST', '/api/auth/mock', {'username': 'mpoolA' + ts})
call(B, 'POST', '/api/auth/mock', {'username': 'mpoolB' + ts})

_, ma = call(A, 'GET', '/api/market')
_, mb = call(B, 'GET', '/api/market')
la, lb = ma['listings'], mb['listings']
print('A sees %d listings (%d npc), B sees %d (%d npc)'
      % (len(la), sum(1 for x in la if x['npc']), len(lb), sum(1 for x in lb if x['npc'])))
assert all('seller' not in x and 'seller_id' not in x for x in la), 'listing leaks seller'
print('✓ 拍賣匿名：payload 無 seller 欄位')

ids_a = {x['stone_id'] for x in la if x['npc']}
ids_b = {x['stone_id'] for x in lb if x['npc']}
overlap = ids_a & ids_b
assert not overlap, 'pools overlap: %s' % overlap
print('✓ 礦區直送池獨立：A(%d) ∩ B(%d) = 空' % (len(ids_a), len(ids_b)))

# B tries to buy A's pool stone
target = next(x for x in la if x['npc'] and x['ask_price'] <= 9000)
st, body = call(B, 'POST', '/api/market/buy', {'listing_id': target['id']})
assert st >= 400, 'B bought A\'s pool stone!'
print('✓ 別人的池子買不到：B 得到 %s' % body.get('error'))

# A buys it; pool refills
_, me = call(A, 'GET', '/api/me')
st, res = call(A, 'POST', '/api/market/buy', {'listing_id': target['id']})
assert st == 200, res
assert res['chips'] == me['chips'] - target['ask_price'], (res, me, target)
print('✓ A 買下自己的池石：%d → %d (扣 %d)' % (me['chips'], res['chips'], target['ask_price']))

_, ma2 = call(A, 'GET', '/api/market')
npc2 = sum(1 for x in ma2['listings'] if x['npc'])
assert npc2 == 6, 'pool did not refill to 6: %d' % npc2
print('✓ 買完自動補貨：池內仍 %d 顆' % npc2)

_, inv = call(A, 'GET', '/api/inventory')
got = [s for s in inv['stones'] if s['id'] == target['stone_id']]
assert got, 'bought stone missing from warehouse'
print('✓ 買到的石頭已入倉庫（origin=%s）' % got[0].get('origin'))

# player-to-player listing is still visible to others, anonymously
_, shop = call(A, 'GET', '/api/shop')
sid = shop['grades'][0]['items'][0]['id']
call(A, 'POST', '/api/shop/buy', {'stone_id': sid})
call(A, 'POST', '/api/market/list', {'stone_id': sid, 'ask_price': 1000})
_, mb2 = call(B, 'GET', '/api/market')
seen = [x for x in mb2['listings'] if x['stone_id'] == sid]
assert seen and 'seller' not in seen[0], 'player listing not anonymous'
print('✓ 玩家掛單：B 看得到，但不知賣家是誰')
print('\n競標場 LIVE CHECKS PASSED')
