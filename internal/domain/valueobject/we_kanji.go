package valueobject

import (
	"strings"
	"unicode/utf8"
)

// WeKanji はWORLD'S ENDのカテゴリ漢字を表す値オブジェクト
type WeKanji string

// NewWeKanji はWeKanjiを生成します（1文字に正規化）
func NewWeKanji(s string) WeKanji {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 複数文字の場合は最初の1文字のみ使用
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return ""
	}
	return WeKanji(string(r))
}

// ReconstructWeKanji はDBから読み込んだ文字列からWeKanjiを復元します
func ReconstructWeKanji(s string) WeKanji {
	return WeKanji(s)
}

// String はWeKanjiを文字列として返します
func (w WeKanji) String() string {
	return string(w)
}

// StringPtr はWeKanjiを*stringとして返します（空の場合はnil）
func (w WeKanji) StringPtr() *string {
	if w == "" {
		return nil
	}
	s := string(w)
	return &s
}

// IsEmpty はWeKanjiが空かどうかを返します
func (w WeKanji) IsEmpty() bool {
	return w == ""
}
