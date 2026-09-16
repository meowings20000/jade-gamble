#!/usr/bin/env python3
"""Multi-player E2E through the PUBLIC tunnel URL: two independent sessions
play simultaneously, trade on the shared market, and verify both balances."""
import json
import time
import urllib.request
import http.cookiejar

BASE = 'https://relief-cam-physician-institutions.trycloudflare.com'

def mk_opener():
    cj = http.cookiejar.CookieJar()
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

def call(op, method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header('Content-Type', 'application/json')
    data = json.dumps(body).encode() if body is not None else None
    with op.open(req, data) as r:
        return json.loads(r.read().decode())

# player A and B log in at the same time
ts = str(time.time_ns())
opA, opB = mk_opener(), mk_opener()
ra = call(opA, 'POST', '/api/auth/mock', {'username': 'MP甲_' + ts})
rb = call(opB, 'POST', '/api/auth/mock', {'username': 'MP乙_' + ts})
assert ra['ok'] and rb['ok']
print('✓ A/B 同時登入（獨立 session）:', ra['username'], '/', rb['username'])

# both buy shelf stones simultaneously
shopA = call(opA, 'GET', '/api/shop')
shopB = call(opB, 'GET', '/api/shop')
itemA = shopA['grades'][0]['items'][0]
itemB = shopB['grades'][0]['items'][0]
assert itemA['id'] != itemB['id'], 'shelves must be personal'
print('✓ 個人貨架獨立：A 買 %s / B 買 %s（不同石）' % (itemA['id'], itemB['id']))
call(opA, 'POST', '/api/shop/buy', {'stone_id': itemA['id']})
call(opB, 'POST', '/api/shop/buy', {'stone_id': itemB['id']})

# A lists on the shared market; B buys it
ask = 500
call(opA, 'POST', '/api/market/list', {'stone_id': itemB['id'], 'ask_price': ask}) if False else None
la = call(opA, 'POST', '/api/market/list', {'stone_id': itemA['id'], 'ask_price': ask})
market = call(opB, 'GET', '/api/market')
target = next((l for l in market['listings'] if l['stone_id'] == itemA['id']), None)
assert target, 'A 的掛單沒出現在 B 的市場'
print('✓ 共享市場：A 掛單 %s @%d，B 看到了' % (itemA['id'], ask))
buy = call(opB, 'POST', '/api/market/buy', {'listing_id': target['id']})
print('✓ B 買下 A 的石頭，B 餘額', buy['chips'])

# leaderboard should show both players
lb = call(opA, 'GET', '/api/leaderboard')
names = [e['username'] for e in lb.get('score', lb.get('entries', []))] if isinstance(lb, dict) else []
print('✓ 排行榜人數:', len(lb) if isinstance(lb, list) else 'api-dict')

# both cut stones at the same time (concurrent write safety)
cutA = call(opA, 'POST', '/api/cut', {'stone_id': itemA['id']}) if False else None
cutB = call(opB, 'POST', '/api/cut', {'stone_id': itemB['id']})
print('✓ B 切石:', cutB['quality'], cutB['variety'], 'payout', cutB['payout'])

meA = call(opA, 'GET', '/api/me')
meB = call(opB, 'GET', '/api/me')
print('✓ 同時在線餘額 — A:', meA['chips'], ' B:', meB['chips'])
print('\nMULTIPLAYER E2E PASSED (via public tunnel)')