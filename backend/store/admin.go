package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// 管理員控制臺的資料層（2026-09-16）
//
// 管理員名單放 DB（admins），來源可能是 .env 的 ADMIN_DISCORD_IDS 或
// 「第一個用 Discord 登入的真人」自動 bootstrap。所有管理動作都留 admin_log。

// AdminPlayer: 控制臺的玩家列表一列。
type AdminPlayer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Discord  string `json:"discord_id"`
	Chips    int    `json:"chips"`
	Stones   int    `json:"stones"`
	IsAdmin  bool   `json:"is_admin"`
	IsMock   bool   `json:"is_mock"`
	LastSeen string `json:"last_seen"`
}

// AdminEvent: 活動公告。
type AdminEvent struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	HoursLeft int    `json:"hours_left"`
}

// IsAdmin: 這個 user id 在不在管理員名單裡。
func (s *Store) IsAdmin(uid int) bool {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admins WHERE user_id = ?`, uid).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// AdminCount: 目前有幾個管理員。
func (s *Store) AdminCount() int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	return n
}

// AddAdmin: 加一個管理員（重複呼叫不會出錯）。
func (s *Store) AddAdmin(uid int, note string) error {
	_, err := s.db.Exec(`INSERT INTO admins (user_id, note) VALUES (?,?)
		ON CONFLICT(user_id) DO NOTHING`, uid, note)
	return err
}

// AdminPlayers: 玩家列表（籌碼最多的排前面）。
func (s *Store) AdminPlayers(limit int) ([]AdminPlayer, error) {
	// COALESCE：重建/匯入後若有 NULL，不要讓整個列表掃描失敗
	rows, err := s.db.Query(`SELECT u.id, COALESCE(u.username,''), COALESCE(u.discord_id,''), COALESCE(u.chips,0),
		(SELECT COUNT(*) FROM stones st WHERE st.owner_id = u.id AND st.state = 'owned'),
		CASE WHEN a.user_id IS NULL THEN 0 ELSE 1 END
		FROM users u LEFT JOIN admins a ON a.user_id = u.id
		WHERE COALESCE(u.discord_id,'') NOT LIKE 'bot:%'
		ORDER BY u.chips DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminPlayer{}
	for rows.Next() {
		var p AdminPlayer
		var isAdmin int
		if err := rows.Scan(&p.ID, &p.Name, &p.Discord, &p.Chips, &p.Stones, &isAdmin); err != nil {
			return nil, err
		}
		p.IsAdmin = isAdmin == 1
		p.IsMock = len(p.Discord) > 5 && p.Discord[:5] == "mock:"
		out = append(out, p)
	}
	return out, rows.Err()
}

// AdminStats: 營運數字。
func (s *Store) AdminStats() (map[string]any, error) {
	stats := map[string]any{}
	pairs := []struct {
		key string
		q   string
	}{
		{"players", `SELECT COUNT(*) FROM users`},
		{"real_players", `SELECT COUNT(*) FROM users WHERE discord_id NOT LIKE 'mock:%'`},
		{"total_chips", `SELECT COALESCE(SUM(chips),0) FROM users`},
		{"stones_owned", `SELECT COUNT(*) FROM stones WHERE state = 'owned'`},
		{"stones_cut", `SELECT COUNT(*) FROM stones WHERE state = 'used'`},
		{"listings_open", `SELECT COUNT(*) FROM listings WHERE sold = 0`},
		{"bot_buys", `SELECT COUNT(*) FROM bot_buys`},
		{"bot_buys_24h", `SELECT COUNT(*) FROM bot_buys WHERE created_at >= datetime('now','-1 day')`},
		{"polish_running", `SELECT COUNT(*) FROM polish_progress WHERE alive = 1`},
		{"transfers_open", `SELECT COUNT(*) FROM transfers WHERE status = 'pending'`},
	}
	for _, p := range pairs {
		var n int
		if err := s.db.QueryRow(p.q).Scan(&n); err != nil {
			return nil, err
		}
		stats[p.key] = n
	}
	return stats, nil
}

// GrantAllTx: 全服發籌碼，回傳受惠人數。
func (s *Store) GrantAllTx(tx *sql.Tx, amount int) (int, error) {
	res, err := tx.Exec(`UPDATE users SET chips = chips + ?`, amount)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteMockUsersTx: 清掉測試帳號（discord_id 以 mock: 開頭）。
func (s *Store) DeleteMockUsersTx(tx *sql.Tx) (int, error) {
	var ids []int
	rows, err := tx.Query(`SELECT id FROM users WHERE discord_id LIKE 'mock:%'`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		return 0, nil
	}
	for _, id := range ids {
		for _, q := range []string{
			`DELETE FROM stones WHERE owner_id = ?`,
			`DELETE FROM sessions WHERE user_id = ?`,
			`DELETE FROM shelf_items WHERE user_id = ?`,
			`DELETE FROM npc_pool WHERE user_id = ?`,
			`DELETE FROM listings WHERE seller_id = ?`,
			`DELETE FROM discoveries WHERE user_id = ?`,
			`DELETE FROM kbd_progress WHERE user_id = ?`,
			`DELETE FROM admins WHERE user_id = ?`,
		} {
			if _, err := tx.Exec(q, id); err != nil && !isMissingTable(err) {
				return 0, err
			}
		}
		if _, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func isMissingTable(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "no such table") || errors.Is(err, sql.ErrNoRows))
}

// LogAdminActionTx: 管理動作留痕。
func (s *Store) LogAdminActionTx(tx *sql.Tx, adminID int, action string, targetID, amount int, note string) error {
	_, err := tx.Exec(`INSERT INTO admin_log (admin_id, action, target_id, amount, note) VALUES (?,?,?,?,?)`,
		adminID, action, targetID, amount, note)
	return err
}

// CreateEventTx: 發佈活動公告（hours 小時後自動過期）。
func (s *Store) CreateEventTx(tx *sql.Tx, adminID int, title, body string, hours int) (int, error) {
	res, err := tx.Exec(`INSERT INTO events (admin_id, title, body, expires_at)
		VALUES (?,?,?, datetime('now', ?))`, adminID, title, body, fmt.Sprintf("+%d hours", hours))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// ActiveEvents: 還沒過期的活動（新到舊）。
func (s *Store) ActiveEvents() ([]AdminEvent, error) {
	rows, err := s.db.Query(`SELECT id, title, body, created_at, expires_at,
		CAST(ROUND((julianday(expires_at) - julianday('now')) * 24) AS INTEGER)
		FROM events WHERE active = 1 AND expires_at > datetime('now')
		ORDER BY id DESC LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminEvent{}
	for rows.Next() {
		var e AdminEvent
		if err := rows.Scan(&e.ID, &e.Title, &e.Body, &e.CreatedAt, &e.ExpiresAt, &e.HoursLeft); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CloseEvent: 把活動下架。
func (s *Store) CloseEvent(id int) error {
	_, err := s.db.Exec(`UPDATE events SET active = 0 WHERE id = ?`, id)
	return err
}

// PurgeMockUsers: 清掉所有測試帳號（mock:）與 bot 帳號、以及他們的相關資料。
// 只在容器內執行（單一寫入者），不要從主機直接改 DB——那會弄壞 SQLite。
func (s *Store) PurgeMockUsers() (map[string]int, error) {
	out := map[string]int{}
	var ids []int
	rows, err := s.db.Query(`SELECT id FROM users WHERE discord_id LIKE 'mock:%' OR discord_id LIKE 'bot:%'`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		return out, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	in := "(" + strings.Join(ph, ",") + ")"
	stmts := []struct {
		key string
		sql string
	}{
		{"sessions", `DELETE FROM sessions WHERE user_id IN ` + in},
		{"stones", `DELETE FROM stones WHERE owner_id IN ` + in},
		{"shelves", `DELETE FROM shelves WHERE user_id IN ` + in},
		{"lit_stones", `DELETE FROM lit_stones WHERE user_id IN ` + in},
		{"listings", `DELETE FROM listings WHERE seller_id IN ` + in},
		{"transfers", `DELETE FROM transfers WHERE from_id IN ` + in + ` OR to_id IN ` + in},
		{"loans", `DELETE FROM loans WHERE user_id IN ` + in},
		{"heist_seats", `DELETE FROM heist_seats WHERE user_id IN ` + in},
		{"users", `DELETE FROM users WHERE id IN ` + in},
	}
	for _, st := range stmts {
		a := args
		if st.key == "transfers" {
			a = append(append([]any{}, args...), args...)
		}
		res, err := s.db.Exec(st.sql, a...)
		if err != nil {
			out[st.key] = -1
			continue
		}
		n, _ := res.RowsAffected()
		out[st.key] = int(n)
	}
	return out, nil
}
