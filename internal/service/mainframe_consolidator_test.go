package service

import (
	"testing"

	"chunisupport-song-batch/internal/domain/difficulty"
)

// TestNormalizer は normalizer 関数のエッジケースを確認
func TestNormalizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 基本ケース
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple title",
			input:    "Hello World",
			expected: "helloworld",
		},
		// 大文字小文字変換
		{
			name:     "uppercase to lowercase",
			input:    "UPPER CASE",
			expected: "uppercase",
		},
		// 空白の削除
		{
			name:     "multiple spaces",
			input:    "Hello   World",
			expected: "helloworld",
		},
		{
			name:     "tabs and newlines",
			input:    "Hello\tWorld\n",
			expected: "helloworld",
		},
		// NFKC正規化: 全角英数字 → 半角
		{
			name:     "fullwidth alphanumeric",
			input:    "ＡＢＣ１２３",
			expected: "abc123",
		},
		{
			name:     "fullwidth space",
			input:    "A　B",
			expected: "ab",
		},
		// 引用符の正規化
		{
			name:     "curly double quotes",
			input:    "\u201CHello\u201D",
			expected: "\"hello\"",
		},
		{
			name:     "curly single quotes",
			input:    "\u2018Hello\u2019",
			expected: "'hello'",
		},
		// 波線の正規化
		{
			name:     "wave dash",
			input:    "A〜B",
			expected: "a~b",
		},
		{
			name:     "fullwidth tilde",
			input:    "A～B",
			expected: "a~b",
		},
		// JSONエスケープシーケンスの処理
		{
			name:     "escaped double quote",
			input:    `\"Hello\"`,
			expected: "\"hello\"",
		},
		{
			name:     "escaped single quote",
			input:    `\'Hello\'`,
			expected: "'hello'",
		},
		// 複合ケース
		{
			name:     "complex title with special chars",
			input:    "「ナイト・オブ・ナイツ」〜Night of Knights～",
			expected: "「ナイト・オブ・ナイツ」~nightofknights~",
		},
		{
			name:     "mixed fullwidth and halfwidth",
			input:    "ＡＢＣａｂｃ123",
			expected: "abcabc123",
		},
		// 制御文字の除去
		{
			name:     "control characters",
			input:    "Hello\x00World",
			expected: "helloworld",
		},
		// Unicode正規化が必要な文字
		{
			name:     "halfwidth katakana",
			input:    "ｶﾀｶﾅ",
			expected: "カタカナ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normalizer(tt.input)
			if result != tt.expected {
				t.Errorf("normalizer(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNormalizer_RealWorldCases は実際の楽曲タイトルを想定したテスト
func TestNormalizer_RealWorldCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Night of Knights variation 1",
			input:    "ナイト・オブ・ナイツ",
			expected: "ナイト・オブ・ナイツ",
		},
		{
			name:     "Title with brackets",
			input:    "「花」 feat. Hatsune Miku",
			expected: "「花」feat.hatsunemiku",
		},
		{
			name:     "Title with fullwidth numbers",
			input:    "ＣＨＵＮＩＴＨＭ",
			expected: "chunithm",
		},
		{
			name:     "Title with mixed wave dashes",
			input:    "Spring～spring〜SPRING",
			expected: "spring~spring~spring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normalizer(tt.input)
			if result != tt.expected {
				t.Errorf("normalizer(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMapMainframeDifficulty は難易度マッピングを確認
func TestMapMainframeDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected int
	}{
		// 短縮形
		{"BAS", 1},
		{"ADV", 2},
		{"EXP", 3},
		{"MAS", 4},
		{"ULT", 5},
		// 完全形
		{"BASIC", 1},
		{"ADVANCED", 2},
		{"EXPERT", 3},
		{"MASTER", 4},
		{"ULTIMA", 5},
		// 小文字
		{"bas", 1},
		{"adv", 2},
		{"exp", 3},
		{"mas", 4},
		{"ult", 5},
		// 空白付き
		{"  BAS  ", 1},
		// 不正な値
		{"UNKNOWN", 0},
		{"", 0},
		{"INVALID", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := difficulty.ParseName(tt.input).Int()
			if result != tt.expected {
				t.Errorf("difficulty.ParseName(%q).Int() = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}
