
import sqlite3
c = sqlite3.connect(r"C:\jade-gamble\data\jade.db", timeout=30)
before = c.execute("SELECT COUNT(*) FROM users WHERE discord_id LIKE 'mock:%'").fetchone()[0]
print("清除前 mock 帳號:", before)
# 先把殘留的關聯列清掉（有些表用 seller_id / user_id）
for t, col in [("listings","seller_id"), ("loans","user_id"), ("transfers","from_id"), ("transfers","to_id"),
               ("stone_log","user_id"), ("lit_stones","user_id"), ("loan_chat","loan_id")]:
    try:
        if t == "loan_chat":
            c.execute("DELETE FROM loan_chat WHERE loan_id IN (SELECT id FROM loans WHERE user_id IN (SELECT id FROM users WHERE discord_id LIKE 'mock:%'))")
        else:
            c.execute(f"DELETE FROM {t} WHERE {col} IN (SELECT id FROM users WHERE discord_id LIKE 'mock:%')")
    except Exception as e:
        print("  skip", t, col, e)
cur = c.execute("DELETE FROM users WHERE discord_id LIKE 'mock:%'")
print("刪除 users 筆數:", cur.rowcount)
c.commit()
after = c.execute("SELECT COUNT(*) FROM users").fetchone()[0]
print("剩餘帳號:", after)
print("剩餘:", list(c.execute("SELECT id, username, chips FROM users ORDER BY chips DESC")))
c.close()
