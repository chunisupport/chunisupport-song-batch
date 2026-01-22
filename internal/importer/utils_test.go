package importer

import (
	"bytes"
	"testing"
)

func TestRemoveBOM(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "UTF-8 BOM",
			input:    []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "UTF-16 BE BOM",
			input:    []byte{0xFE, 0xFF, 'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "UTF-16 LE BOM",
			input:    []byte{0xFF, 0xFE, 'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "No BOM",
			input:    []byte{'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "Empty data",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "Only BOM",
			input:    []byte{0xEF, 0xBB, 0xBF},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeBOM(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("removeBOM() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTrimBOM(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "UTF-8 BOM",
			input:    []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "No BOM",
			input:    []byte{'h', 'e', 'l', 'l', 'o'},
			expected: []byte{'h', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "Empty data",
			input:    []byte{},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimBOM(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("trimBOM() = %v, want %v", result, tt.expected)
			}
		})
	}
}
