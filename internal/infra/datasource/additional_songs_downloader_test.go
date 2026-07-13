package datasource

import (
	"strings"
	"testing"
)

func TestParseCoursesSheet(t *testing.T) {
	d := &AdditionalSongsDownloader{}
	values := [][]string{
		{"id", "title", "class"},
		{"50000", " First Step ", "1"},
		{"50020", "HORIZON Set", "inf"},
		{"50027", "RANDOM", "extra"},
	}

	courses, err := d.parseCoursesSheet(values)
	if err != nil {
		t.Fatalf("parseCoursesSheet() error = %v", err)
	}
	if len(courses) != 3 {
		t.Fatalf("len(courses) = %d, want 3", len(courses))
	}
	if courses[0].ID != "50000" || courses[0].Title != "First Step" || courses[0].Class != "1" {
		t.Errorf("courses[0] = %+v", courses[0])
	}
	if courses[2].Class != "extra" {
		t.Errorf("courses[2].Class = %q, want extra", courses[2].Class)
	}
}

func TestParseCoursesSheetRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name    string
		values  [][]string
		wantErr string
	}{
		{
			name:    "必須項目不足",
			values:  [][]string{{"id", "title", "class"}, {"50000", "First Step"}},
			wantErr: "missing required fields",
		},
		{
			name:    "ID重複",
			values:  [][]string{{"id", "title", "class"}, {"50000", "First Step", "1"}, {"50000", "Other", "2"}},
			wantErr: "duplicate id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AdditionalSongsDownloader{}
			_, err := d.parseCoursesSheet(tt.values)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
