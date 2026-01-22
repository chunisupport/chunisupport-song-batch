package difficulty

import "strings"

// ID は難易度IDを表します。
type ID int

const (
	Unknown   ID = 0
	Basic     ID = 1
	Advanced  ID = 2
	Expert    ID = 3
	Master    ID = 4
	Ultima    ID = 5
	Worldsend ID = 6
)

// ParseName は難易度名からIDに変換します。
// 大文字小文字を区別せず、前後の空白を無視します。
func ParseName(name string) ID {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "BAS", "BASIC":
		return Basic
	case "ADV", "ADVANCED":
		return Advanced
	case "EXP", "EXPERT":
		return Expert
	case "MAS", "MASTER":
		return Master
	case "ULT", "ULTIMA":
		return Ultima
	case "WE", "WORLDSEND", "WORLD'S END":
		return Worldsend
	default:
		return Unknown
	}
}

// Int はIDをintに変換します。
func (d ID) Int() int {
	return int(d)
}

// Int8 はIDをint8に変換します。
func (d ID) Int8() int8 {
	return int8(d)
}
