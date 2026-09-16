package domain

// Quality (种水) tiers for the jade inside a stone.
type Quality int

const (
	Brick    Quality = iota // 砖头料
	Bean                    // 豆种
	OilGreen                // 油青
	Icy                     // 冰种
	Glass                   // 玻璃种
)

var qualityNames = map[Quality]string{
	Brick: "砖头料", Bean: "豆种", OilGreen: "油青种", Icy: "冰种", Glass: "玻璃种",
}

func (q Quality) Name() string   { return qualityNames[q] }
func (q Quality) String() string { return q.Name() }

// QualityMultipliers: base value multiplier per quality tier.
var qualityMult = map[Quality]float64{
	Brick: 0.05, Bean: 0.35, OilGreen: 0.8, Icy: 2.2, Glass: 6.0,
}

func (q Quality) Multiplier() float64 { return qualityMult[q] }

// ColorVariety is the color/variety roll, independent of quality.
type ColorVariety int

const (
	Base          ColorVariety = iota
	Violet                     // 紫罗兰
	BlueWater                  // 蓝水
	WhiteGreen                 // 白底青
	FloatingBlue               // 飘蓝花
	InkGreen                   // 墨翠
	YellowGreen                // 黄加绿
	SpringPurple               // 春带彩
	FortuneThree               // 福禄寿(三彩)
	ImperialGreen              // 帝王绿
)

var varietyNames = map[ColorVariety]string{
	Base:          "底色",
	Violet:        "紫罗兰",
	BlueWater:     "蓝水",
	WhiteGreen:    "白底青",
	FloatingBlue:  "飘蓝花",
	InkGreen:      "墨翠",
	YellowGreen:   "黄加绿",
	SpringPurple:  "春带彩",
	FortuneThree:  "福禄寿",
	ImperialGreen: "帝王绿",
}

// varietyProb is the roll chance for each variety (Base takes the rest).
// Roll happens only for grades Bean and above; Brick is always Base.
var varietyProb = map[ColorVariety]float64{
	Violet:        0.030,
	BlueWater:     0.025,
	WhiteGreen:    0.020,
	FloatingBlue:  0.015,
	InkGreen:      0.012,
	YellowGreen:   0.008,
	SpringPurple:  0.004,
	FortuneThree:  0.0015,
	ImperialGreen: 0.0005,
}

// varietyMult: value multiplier applied on top of quality.
var varietyMult = map[ColorVariety]float64{
	Base:          1.0,
	Violet:        1.8,
	BlueWater:     2.0,
	WhiteGreen:    2.2,
	FloatingBlue:  2.5,
	InkGreen:      3.0,
	YellowGreen:   4.0,
	SpringPurple:  5.0,
	FortuneThree:  8.0,
	ImperialGreen: 15.0,
}

func (v ColorVariety) Name() string        { return varietyNames[v] }
func (v ColorVariety) IsExotic() bool      { return v != Base }
func (v ColorVariety) Multiplier() float64 { return varietyMult[v] }

// CollectionScore points awarded on first discovery of each variety.
var varietyScore = map[ColorVariety]int{
	Violet: 50, BlueWater: 60, WhiteGreen: 70, FloatingBlue: 90, InkGreen: 120,
	YellowGreen: 180, SpringPurple: 250, FortuneThree: 350, ImperialGreen: 500,
}

func (v ColorVariety) CollectionScore() int { return varietyScore[v] }
