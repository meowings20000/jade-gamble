
import sqlite3, glob, os, sys
corrupt = sorted(glob.glob(r"C:\jade-gamble\data\backups\jade_corrupt_*.db"))[-1]
print("來源:", os.path.basename(corrupt))
src = sqlite3.connect("file:" + corrupt.replace("\\", "/") + "?mode=ro", uri=True, timeout=30)
dst = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=30)
dst.row_factory = None
tables = {
    "users": ["id", "username", "discord_id", "avatar", "chips", "created_at"],
    "stones": None, "shelves": None, "market_listings": None, "lit_stones": None,
    "transfers": None, "loans": None, "titles_json": None,
}
restored = {}
for t, cols in tables.items():
    try:
        cur = src.execute(f"SELECT * FROM {t}")
        names = [d[0] for d in cur.description]
        rows = []
        while True:
            try:
                r = cur.fetchone()
            except sqlite3.DatabaseError as e:
                print(f"  {t}: 讀到壞頁，停止（{str(e)[:40]}）"); break
            if r is None:
                break
            rows.append(r)
        if not rows:
            restored[t] = 0; continue
        keep = [c for c in names if c.lower() not in ("exposed",)]
        sql = f"INSERT OR REPLACE INTO {t} ({','.join(keep)}) VALUES ({','.join('?'*len(keep))})"
        idx = [names.index(c) for c in keep]
        n = 0
        for r in rows:
            try:
                dst.execute(sql, [r[i] for i in idx]); n += 1
            except Exception:
                pass
        dst.commit(); restored[t] = n
    except Exception as e:
        restored[t] = f"失敗 {str(e)[:50]}"
print("還原結果:", restored)
print("新 DB 帳號:", dst.execute("SELECT COUNT(*) FROM users").fetchone()[0])
print("新 DB 真人:", dst.execute("SELECT username, chips FROM users WHERE discord_id NOT LIKE 'mock:%' AND discord_id NOT LIKE 'bot:%' ORDER BY chips DESC").fetchall())
dst.close()
