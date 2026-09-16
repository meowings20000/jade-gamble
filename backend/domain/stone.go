package domain

import (
	"fmt"
	"time"
)

// Stone is the sealed truth of one rough stone. Revealed fields track what
// the owner (or market viewers) have seen so far. Server-only, never shipped
// whole to the client before processing.
type Stone struct {
	ID    string
	Seed  uint64 // drives client-side procedural rendering of skin & shape
	Grade ShopGrade
	Price int // shop price paid (or listing price)

	OwnerID int
	State   StoneState

	Quality    Quality
	Variety    ColorVariety
	CrackCells []int // indices of grid cells containing cracks
	CracksDeep bool  // has a life-threatening deep crack
	InkHidden  bool  // InkGreen variety: appears worthless until opened

	// LightHint is the pre-generated flashlight report text (may lie).
	LightHint string
	// LightLieRate is the fraction of lies embedded in LightHint (per grade).
	LightLieRate float64

	// Egg flags (story-driven special stones).
	Egg string // "", "bianhe", "b_fake", "spring"
	// BianheGuaranteed: Bianhe egg forces ImperialGreen+Glass on cut.
	BianheGuaranteed bool

	// Origin: "shop" (bought from shelf) or "classic" (traditional-mode bet stone).
	Origin string

	CreatedAt time.Time
}

// ShopGrade is the shop shelf tier.
type ShopGrade int

const (
	KiloGrade    ShopGrade = iota // 公斤料 (blind)
	FeatureGrade                  // 表现料 (skin clues)
	WindowGrade                   // 开窗料 (window opened)
)

var gradeNames = map[ShopGrade]string{
	KiloGrade:    "公斤料",
	FeatureGrade: "表现料",
	WindowGrade:  "开窗料",
}

func (g ShopGrade) Name() string { return gradeNames[g] }

// Grade economics: target EV multiplier relative to price, and price band.
type GradeEcon struct {
	EVMult  float64 // target expected value / price
	PriceLo int
	PriceHi int
	LieRate float64 // flashlight report lie fraction
}

var gradeEcon = map[ShopGrade]GradeEcon{
	// 三檔定位: 蒙頭便宜、賠率最大（變異數最高）；開窗穩定、賠率小。
	KiloGrade:    {EVMult: 0.973, PriceLo: 300, PriceHi: 1200, LieRate: 0.30},
	FeatureGrade: {EVMult: 0.981, PriceLo: 3000, PriceHi: 15000, LieRate: 0.28},
	WindowGrade:  {EVMult: 0.984, PriceLo: 15000, PriceHi: 150000, LieRate: 0.22},
}

func (g ShopGrade) Econ() GradeEcon { return gradeEcon[g] }

// String for logs.
func (s *Stone) String() string {
	return fmt.Sprintf("Stone(%s %s/%s)", s.ID, s.Quality.Name(), s.Variety.Name())
}
