package domain

import "errors"

var ErrInsufficientChips = errors.New("筹码不足")
var ErrNotFound = errors.New("not found")
var ErrStoneNotOwned = errors.New("石头不在你的仓库")
var ErrInvalidMove = errors.New("invalid move")

// User is a player account (Discord-backed).
type User struct {
	ID              int
	DiscordID       string
	Username        string
	Avatar          string
	Chips           int
	CollectionScore int
	Title           string
	StreakBrick     int
	DailyWinStreak  int
	ReliefUsed      int
	ReliefAt        string
	OldMasterRescue int
	LastLoginDate   string
}

// StoneState lifecycle.
type StoneState string

const (
	StateShop   StoneState = "shop"   // on a shelf, not owned
	StateOwned  StoneState = "owned"  // in a user's warehouse
	StateListed StoneState = "listed" // on the market
	StateSold   StoneState = "sold"   // bought from market
	StateUsed   StoneState = "used"   // processed (cut/scratch/polish finished)
)

// LeaderEntry is one row of a leaderboard.
type LeaderEntry struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Score    int    `json:"score"`
	Extra    int    `json:"extra"`
}

// FameEntry: hall of fame cut.
type FameEntry struct {
	UserID    int          `json:"user_id"`
	Username  string       `json:"username"`
	StoneID   string       `json:"stone_id"`
	Seed      uint64       `json:"seed"`
	Quality   Quality      `json:"quality"`
	Variety   ColorVariety `json:"variety"`
	Payout    int          `json:"payout"`
	CreatedAt string       `json:"created_at"`
}
