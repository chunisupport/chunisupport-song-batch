package util

import "testing"

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		name  string
		input bool
		want  int
	}{
		{
			name:  "true should return 1",
			input: true,
			want:  1,
		},
		{
			name:  "false should return 0",
			input: false,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolToInt(tt.input)
			if got != tt.want {
				t.Errorf("BoolToInt(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
