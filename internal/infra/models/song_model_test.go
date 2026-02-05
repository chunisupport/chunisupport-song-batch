package models_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/models"
)

func TestFromSongEntity(t *testing.T) {
	t.Run("正常系: Songエンティティからモデルに変換", func(t *testing.T) {
		displayID := vo.MustNewDisplayID()
		officialIdx, _ := vo.NewOfficialIdx("12345")
		jacket := vo.NewJacketImage("abc123.png")
		bpm := 120
		releasedAt, _ := vo.ParseReleaseDate("2024-01-15")

		song := entity.NewSong(displayID, "Test Song", "Test Artist", 1, officialIdx, false)
		song.SetJacket(jacket)
		song.SetBPM(bpm)
		song.SetReleasedAt(releasedAt)

		model := models.FromSongEntity(song)

		if model.DisplayID != displayID.String() {
			t.Errorf("DisplayID = %v, want %v", model.DisplayID, displayID.String())
		}
		if model.Title != "Test Song" {
			t.Errorf("Title = %v, want 'Test Song'", model.Title)
		}
		if model.OfficialIdx != "12345" {
			t.Errorf("OfficialIdx = %v, want '12345'", model.OfficialIdx)
		}
		if model.IsWorldsEnd != 0 {
			t.Errorf("IsWorldsEnd = %v, want 0", model.IsWorldsEnd)
		}
	})

	t.Run("正常系: WORLD'S END楽曲", func(t *testing.T) {
		displayID := vo.MustNewDisplayID()
		officialIdx, _ := vo.NewOfficialIdx("12345")

		song := entity.NewSong(displayID, "WE Song", "WE Artist", 1, officialIdx, true)

		model := models.FromSongEntity(song)

		if model.IsWorldsEnd != 1 {
			t.Errorf("IsWorldsEnd = %v, want 1", model.IsWorldsEnd)
		}
	})
}

func TestSongModel_ToSongEntity(t *testing.T) {
	t.Run("正常系: モデルからSongエンティティに変換", func(t *testing.T) {
		bpm := 180
		releasedAt := "2024-01-15"
		model := &models.SongModel{
			ID:          100,
			DisplayID:   "abc123def456",
			Title:       "Test Song",
			Artist:      "Test Artist",
			GenreID:     1,
			OfficialIdx: "12345",
			Jacket:      "jacket123",
			BPM:         &bpm,
			ReleasedAt:  &releasedAt,
			IsWorldsEnd: 0,
			IsDeleted:   0,
		}

		song := model.ToSongEntity()

		if song.ID() != 100 {
			t.Errorf("ID() = %v, want 100", song.ID())
		}
		if song.DisplayID().String() != "abc123def456" {
			t.Errorf("DisplayID() = %v, want 'abc123def456'", song.DisplayID())
		}
		if song.BPM() == nil || *song.BPM() != 180 {
			t.Errorf("BPM() = %v, want 180", song.BPM())
		}
		if song.ReleasedAt().String() != "2024-01-15" {
			t.Errorf("ReleasedAt() = %v, want '2024-01-15'", song.ReleasedAt())
		}
	})
}
