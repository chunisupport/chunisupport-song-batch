package valueobject_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		wantErr  bool
	}{
		{"整数レベル", "13", 13.0, false},
		{"+付きレベル", "14+", 14.5, false},
		{"小数点レベル", "13.5", 13.5, false},
		{"空文字", "", 0, false},
		{"空白のみ", "  ", 0, false},
		{"空白付き", " 14+ ", 14.5, false},
		{"無効な値", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Float64() != tt.expected {
				t.Errorf("ParseLevel() = %v, want %v", got.Float64(), tt.expected)
			}
		})
	}
}

func TestLevel_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		level    valueobject.Level
		expected bool
	}{
		{"ゼロ", valueobject.Level(0), true},
		{"非ゼロ", valueobject.Level(13.5), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsZero(); got != tt.expected {
				t.Errorf("IsZero() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLevel_IsConstUnknown(t *testing.T) {
	tests := []struct {
		name     string
		level    valueobject.Level
		expected bool
	}{
		{"10.0以上", valueobject.Level(10.0), true},
		{"14.5", valueobject.Level(14.5), true},
		{"9.9", valueobject.Level(9.9), false},
		{"0", valueobject.Level(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsConstUnknown(); got != tt.expected {
				t.Errorf("IsConstUnknown() = %v, want %v", got, tt.expected)
			}
		})
	}
}
