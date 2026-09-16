
import sqlite3, json, urllib.request
db = r"C:\jade-gamble\data\jade.db"
c = sqlite3.connect(db, timeout=30)
mock_ids = [r[0] for r in c.execute("SELECT id FROM users WHERE discord_id LIKE 'mock:%'")]
real = list(c.execute("SELECT id, username, chips, discord_id FROM users WHERE discord_id NOT LIKE 'mock:%'"))
print(f"測試帳號 {len(mock_ids)} 個；真人帳號 {len(real)} 個：")
for r in real: print("   保留 →", r[1], r[2], "籌碼")
tables = ["stones","sessions","shelves","npc_pool","listings","discovered","stone_log","loans","transfers",
          "inventory_items","active_buffs","lit_stones","scratch_progress","polish_progress","loan_chat","hall_of_fame"]
ph = ",".join("?"*len(mock_ids)) if mock_ids else None
removed = {}
if mock_ids:
    for t in tables:
        cols = [r[1] for r in c.execute(f"PRAGMA table_info({t})")]
        col = next((x for x in ("user_id","owner_id","seller_id","from_id","to_id","admin_id","seller_id") if x in cols), None)
        if not col: continue
        try:
            cur = c.execute(f"DELETE FROM {t} WHERE {col} IN ({ph})", mock_ids)
            if cur.rowcount > 0: removed[t] = cur.rowcount
        except Exception as e:
            print("  skip", t, e)
    c.execute(f"DELETE FROM admins WHERE user_id IN ({ph})", mock_ids)
    c.execute(f"DELETE FROM users WHERE id IN ({ph})", mock_ids)
c.commit()
print("已刪除：", removed, "users:", len(mock_ids))
print("剩餘帳號:", list(c.execute("SELECT COUNT(*) FROM users"))[0][0])
c.close()
