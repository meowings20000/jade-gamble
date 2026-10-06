package store

import (
	"time"
)

// ═══════════ 集中活動（server-wide buff）═══════════
// user 2026-10-06「那8個活動把他做掉」：admin 開關、全服生效、時限自動關。
// 8 種：
//  1 polish_luck   爆率減半（彩蛋機率×2 ≈ 切石爽感）
//  2 salvage_30    磨崩返還 10% → 30%
//  3 market_fee_0  貨架上架費 5% → 0%
//  4 checkin_x2    簽到雙倍（10萬→20萬 /VIP 20→40萬）
//  5 race_prize    賽季排行榜獎池加碼（現有 leaderboards，領獎 ×2）
//  6 special_stone 限定石頭：生成時特殊品種機率 boost
//  7 heist_bonus   奪寶獎池再 ×1.5（疊在 7.5 上）
//  8 ai_weekend    AI 開市門檻 ×0.5（週末限定用）

type ServerEvent struct {
	Key       string `json:"key"`
	ExpiresAt string `json:"expires_at"`
}

// ServerEventActivate: 開（或續期）一個活動。hours <= 0 = 永久直到手動關。
func (s *Store) ServerEventActivate(key string, hours int, adminID int) error {
	if hours <= 0 {
		hours = 24 * 7
	}
	exp := time.Now().Add(time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`INSERT INTO server_events (key, expires_at, admin_id) VALUES (?,?,?)
		ON CONFLICT(key) DO UPDATE SET expires_at=excluded.expires_at, admin_id=excluded.admin_id`,
		key, exp, adminID)
	if err == nil {
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", exp, time.Local)
		s.eventMu.Lock()
		s.events[key] = t
		s.eventMu.Unlock()
	}
	return err
}

// ServerEventDeactivate: 手動關。
func (s *Store) ServerEventDeactivate(key string) error {
	_, err := s.db.Exec(`DELETE FROM server_events WHERE key=?`, key)
	if err == nil {
		s.eventMu.Lock()
		delete(s.events, key)
		s.eventMu.Unlock()
	}
	return err
}

// loadServerEventCache 啟動時把活動狀態載入記憶體。
// 活動效果會在 transaction 內查詢；若再查單連線 SQLite，會等待自己而永久卡死。
func (s *Store) loadServerEventCache() error {
	rows, err := s.db.Query(`SELECT key, expires_at FROM server_events`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var key, exp string
		if err := rows.Scan(&key, &exp); err != nil {
			return err
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", exp, time.Local)
		if err == nil && now.Before(t) {
			s.events[key] = t
		}
	}
	return rows.Err()
}

// ServerEventActive: 純記憶體查詢，安全供 DB transaction 內呼叫。
func (s *Store) ServerEventActive(key string) bool {
	s.eventMu.RLock()
	exp, ok := s.events[key]
	s.eventMu.RUnlock()
	if !ok || time.Now().After(exp) {
		if ok {
			s.eventMu.Lock()
			delete(s.events, key)
			s.eventMu.Unlock()
		}
		return false
	}
	return true
}

// ServerEventList: admin 面板用（列出所有 active）。
func (s *Store) ServerEventList() ([]ServerEvent, error) {
	rows, err := s.db.Query(`SELECT key, expires_at FROM server_events ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ServerEvent{}
	now := time.Now()
	for rows.Next() {
		var k, exp string
		if err := rows.Scan(&k, &exp); err != nil {
			continue
		}
		t, e := time.ParseInLocation("2006-01-02 15:04:05", exp, time.Local)
		if e != nil || now.After(t) {
			continue
		}
		out = append(out, ServerEvent{Key: k, ExpiresAt: exp})
	}
	return out, rows.Err()
}

// ═══ 生效點的查詢便捷函式 ═══

// EventCheckinX2: 簽到雙倍？
func (s *Store) EventCheckinX2() bool { return s.ServerEventActive("checkin_x2") }

// EventSalvage30: 磨崩返還 30%？
func (s *Store) EventSalvage30() bool { return s.ServerEventActive("salvage_30") }

// EventMarketFree: 上架費免 5%？
func (s *Store) EventMarketFree() bool { return s.ServerEventActive("market_fee_0") }

// EventHeistBoost: 奪寶再 ×1.5？
func (s *Store) EventHeistBoost() bool { return s.ServerEventActive("heist_bonus") }

// EventPolishLuck: 切石爽感（彩蛋 ×2）？
func (s *Store) EventPolishLuck() bool { return s.ServerEventActive("polish_luck") }

// EventSpecialStone: 限定石 boost？
func (s *Store) EventSpecialStone() bool { return s.ServerEventActive("special_stone") }

// EventRacePrize: 排行榜獎金 ×2？
func (s *Store) EventRacePrize() bool { return s.ServerEventActive("race_prize") }

// EventAIWeekend: AI 門檻半價？
func (s *Store) EventAIWeekend() bool { return s.ServerEventActive("ai_weekend") }
