package datasource

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAdditionalSongsParsersKeepEmptyResultsAsJSONArrays(t *testing.T) {
	tests := []struct {
		name      string
		songRows  [][]string
		chartRows [][]string
		weRows    [][]string
	}{
		{
			name:      "header only",
			songRows:  [][]string{{"id", "title", "artist", "genre", "release"}},
			chartRows: [][]string{{"id", "diff", "const"}},
			weRows:    [][]string{{"id", "title", "artist", "genre", "release"}},
		},
		{
			name:      "all data rows are skipped",
			songRows:  [][]string{{"id", "title", "artist", "genre", "release"}, {"", "invalid"}},
			chartRows: [][]string{{"id", "diff", "const"}, {"", "", ""}},
			weRows:    [][]string{{"id", "title", "artist", "genre", "release"}, {"", "invalid"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AdditionalSongsDownloader{}
			songs, err := d.parseSongsSheet(tt.songRows)
			if err != nil {
				t.Fatalf("parseSongsSheet() error = %v", err)
			}
			charts, err := d.parseChartsSheet(tt.chartRows)
			if err != nil {
				t.Fatalf("parseChartsSheet() error = %v", err)
			}
			weCharts, err := d.parseWEChartsSheet(tt.weRows)
			if err != nil {
				t.Fatalf("parseWEChartsSheet() error = %v", err)
			}
			if songs == nil || charts == nil || weCharts == nil {
				t.Fatalf("empty parser results must be non-nil: songs=%v charts=%v weCharts=%v", songs, charts, weCharts)
			}

			data, err := json.Marshal(additionalSongsSheetData{
				Songs:    songs,
				Charts:   charts,
				WECharts: weCharts,
				Courses:  []additionalCourseRow{},
			})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			const want = `{"songs":[],"charts":[],"we_charts":[],"courses":[]}`
			if string(data) != want {
				t.Fatalf("empty results JSON = %s, want %s", data, want)
			}
		})
	}
}

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
