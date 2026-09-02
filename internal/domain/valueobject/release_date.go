package valueobject

import (
	"fmt"
	"strings"
	"time"
)

// ReleaseDate はリリース日を表す値オブジェクト（YYYY-MM-DD形式）
type ReleaseDate string

// ParseReleaseDate は文字列からReleaseDateを生成します
// "YYYY/MM/DD" または "YYYY-MM-DD" 形式を受け付けます
func ParseReleaseDate(s string) (ReleaseDate, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}

	// YYYY/MM/DD → YYYY-MM-DD に正規化
	normalized := strings.ReplaceAll(s, "/", "-")

	// フォーマット検証
	if _, err := time.Parse("2006-01-02", normalized); err != nil {
		return "", fmt.Errorf("invalid release date format %q: %w", s, err)
	}

	return ReleaseDate(normalized), nil
}

// ReconstructReleaseDate はDBから読み込んだ文字列からReleaseDateを復元します
func ReconstructReleaseDate(s string) ReleaseDate {
	return ReleaseDate(s)
}

// String はReleaseDateを文字列として返します（YYYY-MM-DD形式）
func (r ReleaseDate) String() string {
	return string(r)
}

// StringPtr はReleaseDateを*stringとして返します（空の場合はnil）
func (r ReleaseDate) StringPtr() *string {
	if r == "" {
		return nil
	}
	return new(string(r))
}

// IsEmpty はReleaseDateが空かどうかを返します
func (r ReleaseDate) IsEmpty() bool {
	return r == ""
}
