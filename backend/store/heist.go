package store

import (
	"database/sql"
	"jade-gamble/backend/domain"
)

// ---------- 奪寶 heist ----------

// Heist: 一桌。
type Heist struct {
	ID        int
	Grade     int
	Entry     int
	Pot       int
	Progress  int
	Target    int
	Round     int
	Status    string // open / running / done
	CreatedAt string
}

// HeistSeatRow: 一個位子。
type HeistSeatRow struct {
	UserID   int
	Name     string
	Alive    bool
	Action   string
	Target   int
	Entry    int
	Payout   int
	KilledBy int
	Exposed  int // 這個位子知道誰想殺自己（0 = 沒人）
}

// OpenHeist: 找一張還沒開始、同檔位的桌子；沒有就開一張。
func (s *Store) OpenHeistTx(tx *sql.Tx, grade int) (*Heist, error) {
	// 只接手「還沒滿、而且位子都是活著的人」的桌子（避免接手到被清掉帳號留下的空位）
	h, err := scanHeist(tx.QueryRow(`SELECT h.id, h.grade, h.entry, h.pot, h.progress, h.target, h.round, h.status, h.created_at
		FROM heists h
		WHERE h.grade=? AND h.status IN ('open','running')
		  AND (SELECT COUNT(*) FROM heist_seats s WHERE s.heist_id=h.id) < ?
		ORDER BY h.id LIMIT 1`, grade, domain.HeistSeats))
	if err != nil || h != nil {
		return h, err
	}
	entry := domain.HeistEntry(domain.ShopGrade(grade))
	if _, err := tx.Exec(`INSERT INTO heists (grade, entry, pot, progress, target, round, status)
		VALUES (?,?,?,0,?,0,'open')`, grade, entry, 0, domain.HeistTargetFor(domain.ShopGrade(grade))); err != nil {
		return nil, err
	}
	return scanHeist(tx.QueryRow(`SELECT id, grade, entry, pot, progress, target, round, status, created_at
		FROM heists WHERE grade=? AND status IN ('open','running') ORDER BY id DESC LIMIT 1`, grade))
}

func scanHeist(row *sql.Row) (*Heist, error) {
	var h Heist
	err := row.Scan(&h.ID, &h.Grade, &h.Entry, &h.Pot, &h.Progress, &h.Target, &h.Round, &h.Status, &h.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// HeistByID: 取一桌。
func (s *Store) HeistByID(id int) (*Heist, error) {
	return scanHeist(s.db.QueryRow(`SELECT id, grade, entry, pot, progress, target, round, status, created_at
		FROM heists WHERE id=?`, id))
}

// JoinHeistTx: 入座（扣入場費由呼叫端做）。
func (s *Store) JoinHeistTx(tx *sql.Tx, heistID, userID, entry int) error {
	_, err := tx.Exec(`INSERT INTO heist_seats (heist_id, user_id, alive, action, target, entry)
		VALUES (?,?,1,'',0,?)`, heistID, userID, entry)
	return err
}

// HeistSeats: 一桌的位子（含名字）。
func (s *Store) HeistSeats(heistID int) ([]HeistSeatRow, error) {
	rows, err := s.db.Query(`SELECT s.user_id, u.username, s.alive, s.action, s.target, s.entry,
		s.payout, s.killed_by, s.exposed FROM heist_seats s JOIN users u ON u.id = s.user_id
		WHERE s.heist_id=? ORDER BY s.id`, heistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HeistSeatRow{}
	for rows.Next() {
		var r HeistSeatRow
		var alive int
		if err := rows.Scan(&r.UserID, &r.Name, &alive, &r.Action, &r.Target, &r.Entry,
			&r.Payout, &r.KilledBy, &r.Exposed); err != nil {
			return nil, err
		}
		r.Alive = alive == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// MyHeist: 這個人正在進行中的桌子（沒有就 nil）。
func (s *Store) MyHeist(userID int) (*Heist, error) {
	return scanHeist(s.db.QueryRow(`SELECT h.id, h.grade, h.entry, h.pot, h.progress, h.target, h.round, h.status, h.created_at
		FROM heists h JOIN heist_seats s ON s.heist_id = h.id
		WHERE s.user_id=? AND h.status IN ('open','running')
		  AND (SELECT COUNT(*) FROM heist_seats s2 WHERE s2.heist_id=h.id) <= ?
		ORDER BY h.id DESC LIMIT 1`, userID, domain.HeistSeats))
}

// SetHeistActionTx: 記下這一輪的選擇。
func (s *Store) SetHeistActionTx(tx *sql.Tx, heistID, userID int, action string, target int) error {
	_, err := tx.Exec(`UPDATE heist_seats SET action=?, target=? WHERE heist_id=? AND user_id=? AND alive=1`,
		action, target, heistID, userID)
	return err
}

// ResetHeistActionsTx: 新回合開始，清掉上一輪的動作並記下回合開始時間。
func (s *Store) ResetHeistActionsTx(tx *sql.Tx, heistID int) error {
	if _, err := tx.Exec(`UPDATE heist_seats SET action='', target=0 WHERE heist_id=?`, heistID); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE heists SET round_at=datetime('now') WHERE id=?`, heistID)
	return err
}

// HeistRoundAgeSec: 這一輪開始到現在幾秒（沒記到就當成很久＝該結算了）。
func (s *Store) HeistRoundAgeSec(heistID int) float64 {
	var age sql.NullFloat64
	err := s.db.QueryRow(`SELECT (julianday('now') - julianday(round_at)) * 86400.0 FROM heists WHERE id=?`, heistID).Scan(&age)
	if err != nil || !age.Valid {
		return 99999
	}
	return age.Float64
}

// ApplyHeistRoundTx: 把結算結果寫回。
func (s *Store) ApplyHeistRoundTx(tx *sql.Tx, heistID, progress, round int, res domain.HeistRound) error {
	if _, err := tx.Exec(`UPDATE heists SET progress=?, round=?, status='running' WHERE id=?`, progress, round, heistID); err != nil {
		return err
	}
	for uid, killer := range res.Deaths {
		if _, err := tx.Exec(`UPDATE heist_seats SET alive=0, killed_by=? WHERE heist_id=? AND user_id=?`, killer, heistID, uid); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`, res.Looters[killer], killer); err != nil {
			return err
		}
	}
	for victim, attacker := range res.Exposed {
		if _, err := tx.Exec(`UPDATE heist_seats SET exposed=? WHERE heist_id=? AND user_id=?`, attacker, heistID, victim); err != nil {
			return err
		}
	}
	return nil
}

// FinishHeistTx: 結算獎池給倖存者。
func (s *Store) FinishHeistTx(tx *sql.Tx, heistID int, payouts map[int]int) error {
	for uid, amount := range payouts {
		if _, err := tx.Exec(`UPDATE heist_seats SET payout=? WHERE heist_id=? AND user_id=?`, amount, heistID, uid); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE users SET chips = chips + ? WHERE id=?`, amount, uid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE heists SET status='done' WHERE id=?`, heistID); err != nil {
		return err
	}
	return nil
}

// HeistHistory: 我的奪寶紀錄。
func (s *Store) HeistHistory(userID, limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT h.id, h.grade, h.status, h.progress, h.target, s.alive, s.payout,
		s.killed_by, s.exposed, h.created_at
		FROM heist_seats s JOIN heists h ON h.id = s.heist_id
		WHERE s.user_id=? ORDER BY h.id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, grade, progress, target, alive, killedBy, exposed int
		var status, created string
		var payout int
		if err := rows.Scan(&id, &grade, &status, &progress, &target, &alive, &payout, &killedBy, &exposed, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"heist_id": id, "grade": grade, "status": status, "progress": progress, "target": target,
			"alive": alive == 1, "payout": payout, "killed_by": killedBy, "exposed": exposed, "created_at": created,
		})
	}
	return out, rows.Err()
}

// BotUserTx: 取得（或建立）一隻 bot 的帳號，用來佔位子。
func (s *Store) BotUserTx(tx *sql.Tx, key, name string) (int, error) {
	var id int
	err := tx.QueryRow(`SELECT id FROM users WHERE discord_id=?`, "bot:"+key).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := tx.Exec(`INSERT INTO users (username, discord_id, chips, created_at)
		VALUES (?,?,0,datetime('now'))`, name, "bot:"+key)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	return int(newID), err
}

// FillHeistBotsTx: 把桌上的空位補成 bot。
func (s *Store) FillHeistBotsTx(tx *sql.Tx, heistID, entry, want int, bots []domain.HeistBot) (int, error) {
	var have int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM heist_seats WHERE heist_id=?`, heistID).Scan(&have); err != nil {
		return 0, err
	}
	added := 0
	for _, b := range bots {
		if have+added >= want {
			break
		}
		uid, err := s.BotUserTx(tx, b.Key, b.Name)
		if err != nil {
			return added, err
		}
		var exists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM heist_seats WHERE heist_id=? AND user_id=?`, heistID, uid).Scan(&exists); err != nil {
			return added, err
		}
		if exists > 0 {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO heist_seats (heist_id, user_id, alive, action, target, entry)
			VALUES (?,?,1,'',0,?)`, heistID, uid, entry); err != nil {
			return added, err
		}
		added++
	}
	if have+added >= want {
		if _, err := tx.Exec(`UPDATE heists SET status='running' WHERE id=?`, heistID); err != nil {
			return added, err
		}
	}
	return added, nil
}

// HeistBotsIn: 這一桌有哪些 bot（依 discord_id 前綴判斷）＋他們的傾向。
func (s *Store) HeistBotsIn(heistID int) (map[int]domain.HeistBot, error) {
	rows, err := s.db.Query(`SELECT u.id, u.discord_id FROM heist_seats s JOIN users u ON u.id=s.user_id
		WHERE s.heist_id=? AND s.alive=1`, heistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := map[int]string{}
	for rows.Next() {
		var id int
		var did string
		if err := rows.Scan(&id, &did); err != nil {
			return nil, err
		}
		if len(did) > 4 && did[:4] == "bot:" {
			keys[id] = did[4:]
		}
	}
	out := map[int]domain.HeistBot{}
	for id, k := range keys {
		for _, b := range domain.HeistBots {
			if b.Key == k {
				out[id] = b
			}
		}
	}
	return out, nil
}

// HeistQueue: 大廳要顯示的排隊資訊（誰在排、還缺幾個成團）。
type HeistQueue struct {
	ID     int
	Grade  int
	Entry  int
	Status string
	Names  []string
	AgeSec float64
}

// OpenHeistQueues: 目前還在等人的桌子。
func (s *Store) OpenHeistQueues() ([]HeistQueue, error) {
	rows, err := s.db.Query(`SELECT h.id, h.grade, h.entry, h.status,
		(julianday('now') - julianday(h.created_at)) * 86400.0
		FROM heists h WHERE h.status='open'
		ORDER BY h.id DESC LIMIT 12`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HeistQueue{}
	for rows.Next() {
		var q HeistQueue
		if err := rows.Scan(&q.ID, &q.Grade, &q.Entry, &q.Status, &q.AgeSec); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		nr, err := s.db.Query(`SELECT u.username FROM heist_seats s JOIN users u ON u.id = s.user_id
			WHERE s.heist_id = ? ORDER BY s.id`, out[i].ID)
		if err != nil {
			return out, nil
		}
		for nr.Next() {
			var n string
			if err := nr.Scan(&n); err == nil {
				out[i].Names = append(out[i].Names, n)
			}
		}
		nr.Close()
	}
	return out, nil
}

// MarkHeistSeatLeft: 玩家按「離開桌子」——只影響顯示，不動歷史紀錄。
func (s *Store) MarkHeistSeatLeft(heistID, userID int) error {
	_, err := s.db.Exec(`UPDATE heist_seats SET left=1 WHERE heist_id=? AND user_id=?`, heistID, userID)
	return err
}

// MyHeistRecent: 最近（含已結束）的一桌——用來顯示結算結果，不然畫面會直接掉回大廳。
func (s *Store) MyHeistRecent(userID, withinMin int) (*Heist, error) {
	return scanHeist(s.db.QueryRow(`SELECT h.id, h.grade, h.entry, h.pot, h.progress, h.target, h.round, h.status, h.created_at
		FROM heists h JOIN heist_seats s ON s.heist_id = h.id
		WHERE s.user_id=? AND s.left=0
		  AND (h.status IN ('open','running') OR (julianday('now') - julianday(h.created_at)) * 1440.0 < ?)
		ORDER BY h.id DESC LIMIT 1`, userID, withinMin))
}
