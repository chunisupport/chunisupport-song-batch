package valueobject_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewOfficialIdx(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"正常な値", "123", "123", false},
		{"空白付き", " 456 ", "456", false},
		{"空文字", "", "", true},
		{"空白のみ", "   ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewOfficialIdx(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOfficialIdx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("NewOfficialIdx() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestReconstructOfficialIdx(t *testing.T) {
	t.Run("正常系: 文字列からOfficialIdxを復元", func(t *testing.T) {
		original := "123"
		idx := valueobject.ReconstructOfficialIdx(original)
		if idx.String() != original {
			t.Errorf("expected %s, got %s", original, idx.String())
		}
	})
}

func TestOfficialIdx_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		idx      valueobject.OfficialIdx
		expected bool
	}{
		{"空", valueobject.OfficialIdx(""), true},
		{"値あり", valueobject.OfficialIdx("123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.idx.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}
