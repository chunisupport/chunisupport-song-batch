package util

// BoolToInt はboolを0/1のintに変換します。
func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
