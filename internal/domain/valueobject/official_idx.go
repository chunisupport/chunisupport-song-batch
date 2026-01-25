package valueobject

import (
	"fmt"
	"strings"
)

// OfficialIdx は公式楽曲IDを表す値オブジェクト
type OfficialIdx string

// NewOfficialIdx はOfficialIdxを生成します
func NewOfficialIdx(s string) (OfficialIdx, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("official_idx cannot be empty")
	}
	return OfficialIdx(s), nil
}

// ReconstructOfficialIdx はDBから読み込んだ文字列からOfficialIdxを復元します
func ReconstructOfficialIdx(s string) OfficialIdx {
	return OfficialIdx(s)
}

// String はOfficialIdxを文字列として返します
func (o OfficialIdx) String() string {
	return string(o)
}

// IsEmpty はOfficialIdxが空かどうかを返します
func (o OfficialIdx) IsEmpty() bool {
	return o == ""
}
