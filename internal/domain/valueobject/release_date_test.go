package valueobject_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/valueobject"
)

func TestParseReleaseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"YYYY-MM-DD形式", "2024-01-15", "2024-01-15", false},
		{"YYYY/MM/DD形式", "2024/01/15", "2024-01-15", false},
		{"空文字", "", "", false},
		{"空白のみ", "  ", "", false},
		{"空白付き", " 2024-01-15 ", "2024-01-15", false},
		{"無効な形式", "01-15-2024", "", true},
		{"無効な日付", "2024-13-45", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.ParseReleaseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReleaseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.expected {
				t.Errorf("ParseReleaseDate() = %v, want %v", got.String(), tt.expected)
			}
		})
	}
}

func TestReleaseDate_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		date     valueobject.ReleaseDate
		expected bool
	}{
		{"空", valueobject.ReleaseDate(""), true},
		{"値あり", valueobject.ReleaseDate("2024-01-15"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.date.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReleaseDate_StringPtr(t *testing.T) {
	tests := []struct {
		name      string
		date      valueobject.ReleaseDate
		wantNil   bool
		wantValue string
	}{
		{"空はnil", valueobject.ReleaseDate(""), true, ""},
		{"値ありはポインタ", valueobject.ReleaseDate("2024-01-15"), false, "2024-01-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.date.StringPtr()
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
