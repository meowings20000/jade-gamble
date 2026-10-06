package store

// 簽到 + 升級賬戶 + 黑名單（2026-10-06）
//
// 簽到：每天一次，一般 10 萬 / VIP 20 萬（不限累計額度）
// 升級賬戶：一次性花 100 萬 → 永久 VIP
//   perks：簽到 20 萬、專屬頭像框 frame_juema、名字金框、錢莊利率 20%
// 黑名單：debt > 0 = 帶債禁借；還清 → 3 天冷靜期（blacklist_until）

import (
	"database/sql"
	"errors"
	"time"

	"jade-gamble/backend/domain"
)

// todayStr 遊戲日（跟 relief 的 RFC3339 UTC 一致口徑用日期字串）
func todayStr(now time.Time) string { return now.UTC().Format("2006-01-02") }

// IsVIP
func (s *Store) IsVIP(userID int) bool {
	var v int
	_ = s.db.QueryRow(`SELECT vip FROM users WHERE id=?`, userID).Scan(&v)
	return v == 1
}

// CheckinStatus 今天簽了嗎 + 能領多少
func (s *Store) CheckinStatus(userID int, now time.Time) (checked bool, amount int, vip bool) {
	var cd string
	var v int
	row := s.db.QueryRow(`SELECT checkin_date, vip FROM users WHERE id=?`, userID)
	if err := row.Scan(&cd, &v); err != nil {
		return false, domain.CheckinChips, false
	}
	vip = v == 1
	amount = domain.CheckinChips
	if vip {
		amount = domain.VIPCheckinChips
	}
	return cd == todayStr(now), amount, vip
}

// Checkin 簽到（已簽 → ErrChecked；成功 → 加錢、記日期）
var ErrChecked = errors.New("今天已經簽到過了喵")

func (s *Store) Checkin(userID int, now time.Time) (chips int, amount int, err error) {
	if e := s.AIPoolEnsure(); e != nil {
		return 0, 0, e
	}
	checked, amount, _ := s.CheckinStatus(userID, now)
	if checked {
		return 0, amount, ErrChecked
	}
	if _, e := s.db.Exec(`UPDATE users SET chips=chips+?, checkin_date=? WHERE id=?`,
		amount, todayStr(now), userID); e != nil {
		return 0, 0, e
	}
	var c int
	_ = s.db.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&c)
	return c, amount, nil
}

// UpgradeAccount 升級賬戶（花 100 萬；已是 VIP → ErrAlreadyVIP）
var ErrAlreadyVIP = errors.New("已經是升級賬戶了喵")

func (s *Store) UpgradeAccount(userID int, now time.Time) (chips int, err error) {
	var v int
	if err := s.db.QueryRow(`SELECT vip FROM users WHERE id=?`, userID).Scan(&v); err != nil {
		return 0, err
	}
	if v == 1 {
		return 0, ErrAlreadyVIP
	}
	var chipsNow int
	if err := s.db.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&chipsNow); err != nil {
		return 0, err
	}
	if chipsNow < domain.UpgradeCost {
		return 0, ErrInsufficient
	}
	if _, e := s.db.Exec(`UPDATE users SET chips=chips-?, vip=1 WHERE id=?`, domain.UpgradeCost, userID); e != nil {
		return 0, e
	}
	// 送專屬頭像框（重複給不會出錯 —— inventory_items 用 user_id+item_key 查重）
	if _, e := s.db.Exec(`INSERT OR IGNORE INTO inventory_items (user_id, item_key) VALUES (?, 'frame_juema')`, userID); e != nil {
		return 0, e
	}
	var c int
	_ = s.db.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&c)
	return c, nil
}

// BlacklistState 帶債或冷靜期中（禁借）
func (s *Store) BlacklistState(userID int, now time.Time) (debt int, until string, blocked bool) {
	row := s.db.QueryRow(`SELECT debt, blacklist_until FROM users WHERE id=?`, userID)
	var d sql.NullInt64
	var u string
	if err := row.Scan(&d, &u); err != nil {
		return 0, "", false
	}
	debt = int(d.Int64)
	until = u
	if debt > 0 {
		blocked = true
		return
	}
	if u != "" {
		if t, e := time.Parse(time.RFC3339, u); e == nil && t.After(now) {
			blocked = true
		}
	}
	return debt, until, blocked
}

// RepayDebt 還債（部分或全部）→ 清零時自動開 3 天冷靜期
func (s *Store) RepayDebt(userID int, amount int, now time.Time) (debt int, chips int, err error) {
	if amount <= 0 {
		return 0, 0, errors.New("金額要大於 0")
	}
	var d int
	var chipsNow int
	if err := s.db.QueryRow(`SELECT debt, chips FROM users WHERE id=?`, userID).Scan(&d, &chipsNow); err != nil {
		return 0, 0, err
	}
	if d <= 0 {
		return 0, 0, errors.New("沒有欠款喵")
	}
	if chipsNow < amount {
		return 0, 0, ErrInsufficient
	}
	pay := amount
	if pay > d {
		pay = d
	}
	newDebt := d - pay
	if newDebt > 0 {
		if _, e := s.db.Exec(`UPDATE users SET debt=?, chips=chips-? WHERE id=?`, newDebt, pay, userID); e != nil {
			return 0, 0, e
		}
	} else {
		// 還清 → 3 天冷靜期
		until := now.UTC().Add(3 * 24 * time.Hour).Format(time.RFC3339)
		if _, e := s.db.Exec(`UPDATE users SET debt=0, chips=chips-?, blacklist_until=? WHERE id=?`, pay, until, userID); e != nil {
			return 0, 0, e
		}
	}
	var c int
	_ = s.db.QueryRow(`SELECT chips FROM users WHERE id=?`, userID).Scan(&c)
	return newDebt, c, nil
}

// ApplyDebt 沒收後仍欠的錢記到 debt（seizeOverdue 用）
func (s *Store) ApplyDebtTx(tx *Tx, userID int, debt int) error {
	if debt <= 0 {
		return nil
	}
	_, err := tx.Exec(`UPDATE users SET debt=debt+? WHERE id=?`, debt, userID)
	return err
}

// BlacklistBoard 黑名單榜（錢莊頁公開）
func (s *Store) BlacklistBoard(limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id, username, debt FROM users WHERE debt > 0
		ORDER BY debt DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int
		var n string
		var d int
		if err := rows.Scan(&id, &n, &d); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"user_id": id, "username": n, "debt": d})
	}
	return out, rows.Err()
}
// PlayersList 玩家名單（轉賬 pulldown；依活躍排序）
func (s *Store) PlayersList(limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id, username FROM users
		ORDER BY last_login_date DESC, username ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "username": n})
	}
	return out, rows.Err()
}
