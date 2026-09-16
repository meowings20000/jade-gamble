
import sqlite3, time
c = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=20)
print("最近 5 個使用者:", list(c.execute("SELECT id, username, chips FROM users ORDER BY id DESC LIMIT 5")))
print("最近 3 顆石頭:", list(c.execute("SELECT id, owner_id, price FROM stones ORDER BY rowid DESC LIMIT 3")))
print("總石頭數:", c.execute("SELECT COUNT(*) FROM stones").fetchone()[0])
print("journal mode:", c.execute("PRAGMA journal_mode").fetchone())
c.close()
import os
for f in ("jade.db-wal","jade.db-shm"):
    p = os.path.join(r"C:\jade-gamble\data", f)
    print(f, os.path.exists(p) and os.path.getsize(p))
