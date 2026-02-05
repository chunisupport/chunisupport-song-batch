package valueobject_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewDisplayID(t *testing.T) {
	t.Run("正常系: 16文字の16進数IDが生成される", func(t *testing.T) {
		id, err := valueobject.NewDisplayID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(id.String()) != 16 {
			t.Errorf("expected length 16, got %d", len(id.String()))
		}
	})

	t.Run("正常系: 生成されるIDはユニーク", func(t *testing.T) {
		seen := make(map[string]struct{})
		for i := 0; i < 1000; i++ {
			id, err := valueobject.NewDisplayID()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, exists := seen[id.String()]; exists {
				t.Errorf("duplicate ID generated: %s", id.String())
			}
			seen[id.String()] = struct{}{}
		}
	})
}

func TestDisplayID_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		id       valueobject.DisplayID
		expected bool
	}{
		{"空文字", valueobject.DisplayID(""), true},
		{"値あり", valueobject.DisplayID("abc123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReconstructDisplayID(t *testing.T) {
	t.Run("正常系: 文字列からDisplayIDを復元", func(t *testing.T) {
		original := "abc123def456"
		id := valueobject.ReconstructDisplayID(original)
		if id.String() != original {
			t.Errorf("expected %s, got %s", original, id.String())
		}
	})
}
