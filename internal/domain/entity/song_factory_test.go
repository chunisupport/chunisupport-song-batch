package entity_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/entity"
	"chunisupport-song-batch/internal/importer"
)

func TestNewSongFromOfficial(t *testing.T) {
	t.Run("正常系: 通常楽曲", func(t *testing.T) {
		official := &importer.OfficialSong{
			ID:      "12345",
			Title:   " Test Song ",
			Artist:  " Test Artist ",
			Catname: "POPS",
			Image:   "abc123.png",
		}

		song, err := entity.NewSongFromOfficial(official, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if song.Title() != "Test Song" {
			t.Errorf("Title() = %v, want 'Test Song'", song.Title())
		}
		if song.Artist() != "Test Artist" {
			t.Errorf("Artist() = %v, want 'Test Artist'", song.Artist())
		}
		if song.GenreID() != 1 {
			t.Errorf("GenreID() = %v, want 1", song.GenreID())
		}
		if song.OfficialIdx().String() != "12345" {
			t.Errorf("OfficialIdx() = %v, want '12345'", song.OfficialIdx())
		}
		if song.Jacket().String() != "abc123" {
			t.Errorf("Jacket() = %v, want 'abc123'", song.Jacket())
		}
		if song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = true, want false")
		}
		if song.DisplayID().IsEmpty() {
			t.Error("DisplayID() is empty")
		}
	})

	t.Run("正常系: WORLD'S END楽曲（WeKanjiあり）", func(t *testing.T) {
		official := &importer.OfficialSong{
			ID:      "67890",
			Title:   "WE Song",
			Artist:  "WE Artist",
			WeKanji: "狂",
		}

		song, err := entity.NewSongFromOfficial(official, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = false, want true")
		}
	})

	t.Run("正常系: WORLD'S END楽曲（WeStarあり）", func(t *testing.T) {
		official := &importer.OfficialSong{
			ID:     "67890",
			Title:  "WE Song",
			Artist: "WE Artist",
			WeStar: "5",
		}

		song, err := entity.NewSongFromOfficial(official, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = false, want true")
		}
	})

	t.Run("異常系: OfficialIDが空", func(t *testing.T) {
		official := &importer.OfficialSong{
			ID:     "",
			Title:  "Test",
			Artist: "Test",
		}

		_, err := entity.NewSongFromOfficial(official, 1)
		if err == nil {
			t.Error("expected error for empty official ID")
		}
	})
}

func TestNewSongFromAdditional(t *testing.T) {
	t.Run("正常系: BPMとリリース日付き", func(t *testing.T) {
		bpm := 120
		additional := &importer.AdditionalSong{
			ID:      "12345",
			Title:   " Additional Song ",
			Artist:  " Additional Artist ",
			Genre:   "POPS",
			Img:     "jacket.png",
			BPM:     &bpm,
			Release: "2024/01/15",
		}

		song, err := entity.NewSongFromAdditional(additional, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if song.Title() != "Additional Song" {
			t.Errorf("Title() = %v, want 'Additional Song'", song.Title())
		}
		if song.BPM() == nil || *song.BPM() != 120 {
			t.Errorf("BPM() = %v, want 120", song.BPM())
		}
		if song.ReleasedAt().String() != "2024-01-15" {
			t.Errorf("ReleasedAt() = %v, want '2024-01-15'", song.ReleasedAt())
		}
		if song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = true, want false")
		}
	})

	t.Run("正常系: BPMがnil", func(t *testing.T) {
		additional := &importer.AdditionalSong{
			ID:     "12345",
			Title:  "Test",
			Artist: "Test",
			BPM:    nil,
		}

		song, err := entity.NewSongFromAdditional(additional, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if song.BPM() != nil {
			t.Errorf("BPM() = %v, want nil", song.BPM())
		}
	})
}

func TestDetermineIsWorldsEnd(t *testing.T) {
	tests := []struct {
		name     string
		official *importer.OfficialSong
		expected bool
	}{
		{"通常楽曲", &importer.OfficialSong{ID: "1"}, false},
		{"WeKanjiあり", &importer.OfficialSong{ID: "1", WeKanji: "狂"}, true},
		{"WeStarあり", &importer.OfficialSong{ID: "1", WeStar: "5"}, true},
		{"両方あり", &importer.OfficialSong{ID: "1", WeKanji: "狂", WeStar: "5"}, true},
		{"空白のみのWeKanji", &importer.OfficialSong{ID: "1", WeKanji: "  "}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entity.DetermineIsWorldsEnd(tt.official)
			if got != tt.expected {
				t.Errorf("DetermineIsWorldsEnd() = %v, want %v", got, tt.expected)
			}
		})
	}
}
