package valueobject_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestParseWeStarFromOfficial(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"公式値1→星1", "1", 1, false},
		{"公式値3→星2", "3", 2, false},
		{"公式値5→星3", "5", 3, false},
		{"公式値7→星4", "7", 4, false},
		{"公式値9→星5", "9", 5, false},
		{"空文字", "", 0, false},
		{"空白付き", " 5 ", 3, false},
		{"無効な値2", "2", 0, true},
		{"無効な文字", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.ParseWeStarFromOfficial(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWeStarFromOfficial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Int() != tt.expected {
				t.Errorf("ParseWeStarFromOfficial() = %v, want %v", got.Int(), tt.expected)
			}
		})
	}
}

func TestWeStarFromOfficialValue(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
		wantErr  bool
	}{
		{"公式値1→星1", 1, 1, false},
		{"公式値3→星2", 3, 2, false},
		{"公式値5→星3", 5, 3, false},
		{"公式値7→星4", 7, 4, false},
		{"公式値9→星5", 9, 5, false},
		{"公式値0→星0", 0, 0, false},
		{"無効な値2", 2, 0, true},
		{"無効な値10", 10, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.WeStarFromOfficialValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("WeStarFromOfficialValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Int() != tt.expected {
				t.Errorf("WeStarFromOfficialValue() = %v, want %v", got.Int(), tt.expected)
			}
		})
	}
}

func TestNewWeStar(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"星0", 0, false},
		{"星1", 1, false},
		{"星5", 5, false},
		{"星6（範囲外）", 6, true},
		{"負の値", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobject.NewWeStar(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWeStar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWeStar_IntPtr(t *testing.T) {
	tests := []struct {
		name      string
		star      valueobject.WeStar
		wantNil   bool
		wantValue int
	}{
		{"0はnil", valueobject.WeStar(0), true, 0},
		{"1は*1", valueobject.WeStar(1), false, 1},
		{"5は*5", valueobject.WeStar(5), false, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.star.IntPtr()
			if tt.wantNil {
				if got != nil {
					t.Errorf("IntPtr() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("IntPtr() = nil, want %v", tt.wantValue)
				} else if *got != tt.wantValue {
					t.Errorf("IntPtr() = %v, want %v", *got, tt.wantValue)
				}
			}
		})
	}
}
