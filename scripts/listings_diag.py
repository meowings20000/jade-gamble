
import sqlite3, datetime
c = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=20)
rows = list(c.execute("""SELECT id, stone_id, seller_id, ask_price, sold,
  ROUND((julianday('now') - julianday(created_at))*1440) AS age_mins, created_at
  FROM listings WHERE sold=0 ORDER BY age_mins DESC LIMIT 8"""))
print("未售掛單（依放置時間）:")
for r in rows: print("  ", r)
tot = c.execute("SELECT COUNT(*) FROM listings WHERE sold=0 AND seller_id!=0").fetchone()[0]
stale = c.execute("SELECT COUNT(*) FROM listings WHERE sold=0 AND seller_id!=0 AND (julianday('now')-julianday(created_at))*1440 >= 5").fetchone()[0]
print(f"總未售玩家掛單 {tot}，其中放超過 5 分鐘 {stale}")
c.close()
