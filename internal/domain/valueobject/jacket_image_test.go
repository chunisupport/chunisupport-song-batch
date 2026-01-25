package valueobject_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewJacketImage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"拡張子付き", "abc123.png", "abc123"},
		{"拡張子なし", "abc123", "abc123"},
		{"複数ドット", "abc.123.png", "abc.123"},
		{"空文字", "", ""},
		{"空白のみ", "  ", ""},
		{"空白付き", " abc123.jpg ", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueobject.NewJacketImage(tt.input)
			if got.String() != tt.expected {
				t.Errorf("NewJacketImage() = %v, want %v", got.String(), tt.expected)
			}
		})
	}
}

func TestJacketImage_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		jacket   valueobject.JacketImage
		expected bool
	}{
		{"空", valueobject.JacketImage(""), true},
		{"値あり", valueobject.JacketImage("abc123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.jacket.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestJacketImage_NullableString(t *testing.T) {
	tests := []struct {
		name      string
		jacket    valueobject.JacketImage
		wantNil   bool
		wantValue string
	}{
		{"空はnil", valueobject.JacketImage(""), true, ""},
		{"値ありは文字列", valueobject.JacketImage("abc123"), false, "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.jacket.NullableString()
			if tt.wantNil {
				if got != nil {
					t.Errorf("NullableString() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("NullableString() = nil, want %v", tt.wantValue)
				} else if got.(string) != tt.wantValue {
					t.Errorf("NullableString() = %v, want %v", got, tt.wantValue)
				}
			}
		})
	}
}
