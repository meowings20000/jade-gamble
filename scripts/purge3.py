
import sqlite3
c = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=30)
print("LIKE  :", c.execute("SELECT COUNT(*) FROM users WHERE discord_id LIKE 'mock:%'").fetchone()[0])
print("GLOB  :", c.execute("SELECT COUNT(*) FROM users WHERE discord_id GLOB 'mock:*'").fetchone()[0])
print("substr:", c.execute("SELECT COUNT(*) FROM users WHERE substr(discord_id,1,5)='mock:'").fetchone()[0])
print("id>7  :", c.execute("SELECT COUNT(*) FROM users WHERE id > 7").fetchone()[0])
try:
    cur = c.execute("DELETE FROM users WHERE substr(discord_id,1,5)='mock:'")
    print("DELETE substr → rows:", cur.rowcount, "changes():", c.execute("SELECT changes()").fetchone()[0])
    c.commit()
except Exception as e:
    print("DELETE 失敗:", e)
print("剩餘 users:", c.execute("SELECT COUNT(*) FROM users").fetchone()[0])
print("剩餘清單:", list(c.execute("SELECT id, username FROM users")))
print("integrity:", c.execute("PRAGMA quick_check").fetchone())
c.close()
