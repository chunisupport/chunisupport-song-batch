package valueobject_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewWeKanji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"1文字", "狂", "狂"},
		{"2文字以上は1文字に切り詰め", "狂気", "狂"},
		{"空文字", "", ""},
		{"空白のみ", "  ", ""},
		{"空白付き", " 狂 ", "狂"},
		{"ASCII文字", "A", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueobject.NewWeKanji(tt.input)
			if got.String() != tt.expected {
				t.Errorf("NewWeKanji() = %v, want %v", got.String(), tt.expected)
			}
		})
	}
}

func TestWeKanji_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		kanji    valueobject.WeKanji
		expected bool
	}{
		{"空", valueobject.WeKanji(""), true},
		{"値あり", valueobject.WeKanji("狂"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kanji.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWeKanji_StringPtr(t *testing.T) {
	tests := []struct {
		name      string
		kanji     valueobject.WeKanji
		wantNil   bool
		wantValue string
	}{
		{"空はnil", valueobject.WeKanji(""), true, ""},
		{"値ありはポインタ", valueobject.WeKanji("狂"), false, "狂"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kanji.StringPtr()
			if tt.wantNil {
				if got != nil {
					t.Errorf("StringPtr() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("StringPtr() = nil, want %v", tt.wantValue)
				} else if *got != tt.wantValue {
					t.Errorf("StringPtr() = %v, want %v", *got, tt.wantValue)
				}
			}
		})
	}
}
