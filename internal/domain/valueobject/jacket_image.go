package valueobject

import "strings"

// JacketImage はジャケット画像のハッシュを表す値オブジェクト
type JacketImage string

// NewJacketImage はJacketImageを生成します（拡張子を除去）
func NewJacketImage(s string) JacketImage {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 拡張子を除去
	if dotIndex := strings.LastIndex(s, "."); dotIndex != -1 {
		s = s[:dotIndex]
	}
	return JacketImage(s)
}

// ReconstructJacketImage はDBから読み込んだ文字列からJacketImageを復元します
func ReconstructJacketImage(s string) JacketImage {
	return JacketImage(s)
}

// String はJacketImageを文字列として返します
func (j JacketImage) String() string {
	return string(j)
}

// NullableString はJacketImageをnullable文字列として返します
// 空の場合はnilを返します（DB挿入時に使用）
func (j JacketImage) NullableString() any {
	if j == "" {
		return nil
	}
	return string(j)
}

// IsEmpty はJacketImageが空かどうかを返します
func (j JacketImage) IsEmpty() bool {
	return j == ""
}
