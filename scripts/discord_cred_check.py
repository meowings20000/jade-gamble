#!/usr/bin/env python3
"""Discord OAuth 憑證檢查器 — 確認 Client ID + Secret 是同一組、且能用。

用法:
    python scripts/discord_cred_check.py                 # 讀 .env
    python scripts/discord_cred_check.py <id> <secret>   # 直接測一組

判讀:
    invalid_grant  → 憑證正確（Discord 已認證你，只是 code 是假的）
    invalid_client → 憑證配對錯誤（ID 與 Secret 不屬於同一個應用，或 Secret 已被重設）
"""
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

API = 'https://discord.com/api/v10'


def http(url, data=None, headers=None, method=None):
    req = urllib.request.Request(url, data=data, method=method or ('POST' if data else 'GET'))
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    req.add_header('User-Agent', 'DiscordBot (https://example.com, 1.0)')
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            return r.status, r.read().decode('utf-8', 'replace')
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', 'replace')
    except Exception as e:  # noqa: BLE001
        return -1, str(e)


def load_env():
    env = Path(__file__).resolve().parent.parent / '.env'
    if not env.exists():
        return None, None, None
    t = env.read_text(encoding='utf-8')
    g = lambda k: (re.search(rf'{k}=(\S+)', t) or [None, None])[1]  # noqa: E731
    return g('DISCORD_CLIENT_ID'), g('DISCORD_CLIENT_SECRET'), g('DISCORD_REDIRECT_URI')


def main():
    if len(sys.argv) >= 3:
        cid, sec = sys.argv[1], sys.argv[2]
        red = 'https://jade.meowmeow12245ouo.dpdns.org/api/auth/discord/callback'
    else:
        cid, sec, red = load_env()
    if not cid or not sec:
        print('✗ 沒有 Client ID / Secret（.env 缺值，或參數沒帶）')
        return 1

    st, body = http(f'{API}/applications/{cid}/rpc')
    name = ''
    if st == 200:
        m = re.search(r'"name"\s*:\s*"([^"]+)"', body)
        if m:
            # Discord 會把非 ASCII 轉成 \uXXXX
            name = m.group(1).encode().decode('unicode_escape')
    print(f'Client ID {cid} → {name or "(查不到這個應用)"}')

    form = urllib.parse.urlencode({
        'client_id': cid, 'client_secret': sec, 'grant_type': 'authorization_code',
        'code': 'probe-not-a-real-code', 'redirect_uri': red}).encode()
    st, body = http(f'{API}/oauth2/token', data=form,
                    headers={'Content-Type': 'application/x-www-form-urlencoded'})
    print(f'憑證探測 → HTTP {st} {body.strip()[:160]}')
    if '"invalid_grant"' in body:
        print('✓ 憑證正確（ID 與 Secret 配對成功）')
        return 0
    if '"invalid_client"' in body:
        print('✗ 憑證配對錯誤：Secret 不屬於這個應用，或已被重設')
        print('  → Developer Portal 開「該 Client ID 的應用」→ OAuth2 → Reset Secret')
        return 1
    print('? 非預期回應，請看上面內容')
    return 1


if __name__ == '__main__':
    raise SystemExit(main())
