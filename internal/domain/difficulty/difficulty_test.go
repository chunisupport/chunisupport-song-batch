package difficulty

import "testing"

func TestParseName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ID
	}{
		{"basic lowercase", "basic", Basic},
		{"basic uppercase", "BASIC", Basic},
		{"basic abbreviation", "BAS", Basic},
		{"basic with spaces", "  basic  ", Basic},
		{"advanced lowercase", "advanced", Advanced},
		{"advanced uppercase", "ADVANCED", Advanced},
		{"advanced abbreviation", "ADV", Advanced},
		{"expert lowercase", "expert", Expert},
		{"expert uppercase", "EXPERT", Expert},
		{"expert abbreviation", "EXP", Expert},
		{"master lowercase", "master", Master},
		{"master uppercase", "MASTER", Master},
		{"master abbreviation", "MAS", Master},
		{"ultima lowercase", "ultima", Ultima},
		{"ultima uppercase", "ULTIMA", Ultima},
		{"ultima abbreviation", "ULT", Ultima},
		{"worldsend lowercase", "worldsend", Worldsend},
		{"worldsend uppercase", "WORLDSEND", Worldsend},
		{"worldsend abbreviation", "WE", Worldsend},
		{"worldsend with apostrophe", "WORLD'S END", Worldsend},
		{"unknown", "invalid", Unknown},
		{"empty", "", Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseName(tt.input)
			if got != tt.want {
				t.Errorf("ParseName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestID_Int(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want int
	}{
		{"Unknown", Unknown, 0},
		{"Basic", Basic, 1},
		{"Advanced", Advanced, 2},
		{"Expert", Expert, 3},
		{"Master", Master, 4},
		{"Ultima", Ultima, 5},
		{"Worldsend", Worldsend, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.Int()
			if got != tt.want {
				t.Errorf("ID(%d).Int() = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestID_Int8(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want int8
	}{
		{"Unknown", Unknown, 0},
		{"Basic", Basic, 1},
		{"Advanced", Advanced, 2},
		{"Expert", Expert, 3},
		{"Master", Master, 4},
		{"Ultima", Ultima, 5},
		{"Worldsend", Worldsend, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.Int8()
			if got != tt.want {
				t.Errorf("ID(%d).Int8() = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
