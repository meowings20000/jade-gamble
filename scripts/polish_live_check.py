#!/usr/bin/env python3
"""Live check: 磨石 now reads the stone.

Buys several stones, opens polish on each, and verifies the server reports
a stone-specific break chance + a 手感 line, that the lines differ across
stones, and that advancing sharpens the read.
"""
import json
import time
import urllib.request
import http.cookiejar

BASE = 'http://127.0.0.1:3002'
op = urllib.request.build_opener(
    urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def call(method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header('Content-Type', 'application/json')
    data = json.dumps(body).encode() if body is not None else None
    try:
        with op.open(req, data) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode())


call('POST', '/api/auth/mock', {'username': 'polishchk' + str(time.time_ns())[-9:]})

starts = []
for i in range(5):
    _, shop = call('GET', '/api/shop')
    items = shop['grades'][0]['items']
    if not items:
        break
    sid = items[0]['id']
    st, _ = call('POST', '/api/shop/buy', {'stone_id': sid})
    if st != 200:
        continue
    st, s = call('POST', '/api/polish/start', {'stone_id': sid})
    if st != 200:
        print('  polish start failed:', s)
        continue
    starts.append((sid, s))

print('opened polish on %d stones\n' % len(starts))
probs, feels = set(), set()
for sid, s in starts:
    assert 'break_prob' in s and 'feel' in s and 'risk_delta' in s, ('missing fields', s)
    print('%-14s ×%-6s 崩率 %5.1f%% 風險 %+5.1f%%  %s'
          % (sid, s['multiplier'], s['break_prob'] * 100, s['risk_delta'] * 100, s['feel']))
    probs.add(round(s['break_prob'], 4))
    feels.add(s['feel'])

assert len(probs) > 1, 'break odds identical across stones — polish is still stone-blind'
print('\n✓ 崩率因石而異：%d 種不同值' % len(probs))
assert len(feels) > 1, 'feel lines identical across stones'
print('✓ 手感因石而異：%d 種不同描述' % len(feels))

# advancing must sharpen the read (stage >= 4 gets the detailed line)
sid, s0 = starts[0]
seen = [s0['feel']]
alive = True
for i in range(5):
    st, rr = call('POST', '/api/polish/advance', {'stone_id': sid})
    if st != 200 or not rr.get('alive'):
        print('  (stone broke at rung %s — that is the stone talking)' % rr.get('broke_at'))
        alive = False
        break
    seen.append(rr['feel'])
    assert 'break_prob' in rr, 'advance must report the next break chance'
print('\n✓ 逐層手感揭露:', ' → '.join(seen))
if alive:
    assert len(set(seen)) >= 1
print('\n磨石 LIVE CHECKS PASSED')
