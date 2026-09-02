package valueobject

import (
	"fmt"
	"strconv"
	"strings"
)

// WeStar はWORLD'S ENDの星数を表す値オブジェクト（1-5）
type WeStar int

// ParseWeStarFromOfficial は公式データの星表記からWeStarを生成します
// 公式値: 1,3,5,7,9 → 実際の星数: 1,2,3,4,5
func ParseWeStarFromOfficial(s string) (WeStar, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid we_star format %q: %w", s, err)
	}
	return WeStarFromOfficialValue(v)
}

// WeStarFromOfficialValue は公式の数値からWeStarを生成します
// 公式値: 1,3,5,7,9 → 実際の星数: 1,2,3,4,5
func WeStarFromOfficialValue(officialValue int) (WeStar, error) {
	switch officialValue {
	case 1:
		return 1, nil
	case 3:
		return 2, nil
	case 5:
		return 3, nil
	case 7:
		return 4, nil
	case 9:
		return 5, nil
	case 0:
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid official we_star value: %d (expected 1, 3, 5, 7, or 9)", officialValue)
	}
}

// NewWeStar は直接の星数（1-5）からWeStarを生成します
func NewWeStar(star int) (WeStar, error) {
	if star < 0 || star > 5 {
		return 0, fmt.Errorf("we_star must be between 0 and 5, got %d", star)
	}
	return WeStar(star), nil
}

// ReconstructWeStar はDBから読み込んだ値からWeStarを復元します
func ReconstructWeStar(star int) WeStar {
	return WeStar(star)
}

// Int はWeStarをintとして返します
func (w WeStar) Int() int {
	return int(w)
}

// IntPtr はWeStarを*intとして返します（0の場合はnil）
func (w WeStar) IntPtr() *int {
	if w == 0 {
		return nil
	}
	return new(int(w))
}

// IsZero はWeStarが未設定かどうかを返します
func (w WeStar) IsZero() bool {
	return w == 0
}
