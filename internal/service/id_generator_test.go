package service

import (
	"testing"
)

func TestGenerateDisplayID(t *testing.T) {
	id1 := GenerateDisplayID()
	id2 := GenerateDisplayID()

	// 長さをチェック
	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}

	// 一意性をチェック
	if id1 == id2 {
		t.Errorf("Expected different IDs, but got same: %s", id1)
	}

	// 16進数形式をチェック
	for _, c := range id1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("ID contains non-hex character: %c", c)
		}
	}
}
