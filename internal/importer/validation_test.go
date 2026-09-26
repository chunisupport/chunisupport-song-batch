package importer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeImportFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestOfficialImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"id":" 1 ","title":" Song "}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "missing id", content: `[{"title":"Song"}]`},
		{name: "blank title", content: `[{"id":"1","title":" "}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOfficialImporter().Import(writeImportFixture(t, tt.content))
			assertValidationResult(t, err, tt.valid)
		})
	}
}

func TestMainframeImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"title":" Song ","diff":" MAS ","const":14.2}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "missing title", content: `[{"diff":"MAS","const":14.2}]`},
		{name: "unsupported difficulty", content: `[{"title":"Song","diff":"WORLD'S END","const":14.2}]`},
		{name: "non-positive const", content: `[{"title":"Song","diff":"MAS","const":0}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMainframeImporter().Import(writeImportFixture(t, tt.content))
			assertValidationResult(t, err, tt.valid)
		})
	}
}

func TestSt1027ImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `{"songs":[{"meta":{"official_id":"1"}}]}`, valid: true},
		{name: "missing songs", content: `{}`},
		{name: "null songs", content: `{"songs":null}`},
		{name: "empty songs", content: `{"songs":[]}`},
		{name: "missing official id", content: `{"songs":[{"meta":{}}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSt1027Importer().Import(writeImportFixture(t, tt.content))
			assertValidationResult(t, err, tt.valid)
		})
	}
}

func TestOtogeDbImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"id":" 1 ","title":" Song "}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "non-numeric id", content: `[{"id":"x","title":"Song"}]`},
		{name: "non-positive id", content: `[{"id":"0","title":"Song"}]`},
		{name: "missing title", content: `[{"id":"1"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOtogeDbImporter().Import(writeImportFixture(t, tt.content))
			assertValidationResult(t, err, tt.valid)
		})
	}
}

func TestAdditionalSongsImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "empty arrays are valid", content: `{"songs":[],"charts":[],"we_charts":[],"courses":[]}`, valid: true},
		{name: "missing array", content: `{"songs":[],"charts":[],"we_charts":[]}`},
		{name: "null array", content: `{"songs":[],"charts":[],"we_charts":[],"courses":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAdditionalSongsImporter().Import(writeImportFixture(t, tt.content))
			assertValidationResult(t, err, tt.valid)
		})
	}
}

func assertValidationResult(t *testing.T, err error, valid bool) {
	t.Helper()
	if valid {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
