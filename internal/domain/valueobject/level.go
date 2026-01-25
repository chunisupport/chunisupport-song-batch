package valueobject

import (
	"fmt"
	"strconv"
	"strings"
)

// Level は譜面のレベル（定数）を表す値オブジェクト
type Level float64

// ParseLevel は文字列からLevelを生成します
// "14+" → 14.5, "13" → 13.0 のように変換します
func ParseLevel(s string) (Level, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, "+", ".5")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid level format %q: %w", s, err)
	}
	return Level(v), nil
}

// NewLevel は数値からLevelを生成します
func NewLevel(v float64) Level {
	return Level(v)
}

// Float64 はLevelをfloat64として返します
func (l Level) Float64() float64 {
	return float64(l)
}

// IsZero はレベルがゼロ（未設定）かどうかを返します
func (l Level) IsZero() bool {
	return l == 0
}

// IsConstUnknown は定数が不明かどうかを判定します（10.0以上の場合は未確定とみなす）
// 注意: この判定基準は公式データの仕様に依存しています
func (l Level) IsConstUnknown() bool {
	return l >= 10.0
}
