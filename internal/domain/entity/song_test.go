package entity_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/entity"
	vo "chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewSong(t *testing.T) {
	t.Run("正常系: 新しいSongが生成される", func(t *testing.T) {
		displayID := vo.MustNewDisplayID()
		officialIdx, _ := vo.NewOfficialIdx("12345")

		song := entity.NewSong(
			displayID,
			"Test Song",
			"Test Artist",
			1,
			officialIdx,
			false,
		)

		if song.DisplayID() != displayID {
			t.Errorf("DisplayID() = %v, want %v", song.DisplayID(), displayID)
		}
		if song.Title() != "Test Song" {
			t.Errorf("Title() = %v, want %v", song.Title(), "Test Song")
		}
		if song.Artist() != "Test Artist" {
			t.Errorf("Artist() = %v, want %v", song.Artist(), "Test Artist")
		}
		if song.GenreID() != 1 {
			t.Errorf("GenreID() = %v, want %v", song.GenreID(), 1)
		}
		if song.OfficialIdx() != officialIdx {
			t.Errorf("OfficialIdx() = %v, want %v", song.OfficialIdx(), officialIdx)
		}
		if song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = true, want false")
		}
		if song.IsDeleted() {
			t.Error("IsDeleted() = true, want false")
		}
		if song.ID() != 0 {
			t.Errorf("ID() = %v, want 0 (not persisted)", song.ID())
		}
	})

	t.Run("正常系: WORLD'S END楽曲として生成", func(t *testing.T) {
		displayID := vo.MustNewDisplayID()
		officialIdx, _ := vo.NewOfficialIdx("67890")

		song := entity.NewSong(
			displayID,
			"WE Song",
			"WE Artist",
			2,
			officialIdx,
			true,
		)

		if !song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = false, want true")
		}
	})
}

func TestSong_SetJacket(t *testing.T) {
	displayID := vo.MustNewDisplayID()
	officialIdx, _ := vo.NewOfficialIdx("12345")

	song := entity.NewSong(displayID, "Test", "Test", 1, officialIdx, false)

	jacket := vo.NewJacketImage("abc123.png")
	song.SetJacket(jacket)

	if song.Jacket() != jacket {
		t.Errorf("Jacket() = %v, want %v", song.Jacket(), jacket)
	}
}

func TestSong_SetBPM(t *testing.T) {
	displayID := vo.MustNewDisplayID()
	officialIdx, _ := vo.NewOfficialIdx("12345")

	song := entity.NewSong(displayID, "Test", "Test", 1, officialIdx, false)

	song.SetBPM(120)

	if song.BPM() == nil {
		t.Error("BPM() = nil, want 120")
	} else if *song.BPM() != 120 {
		t.Errorf("BPM() = %v, want 120", *song.BPM())
	}
}

func TestSong_SetReleasedAt(t *testing.T) {
	displayID := vo.MustNewDisplayID()
	officialIdx, _ := vo.NewOfficialIdx("12345")

	song := entity.NewSong(displayID, "Test", "Test", 1, officialIdx, false)

	releasedAt, _ := vo.ParseReleaseDate("2024-01-15")
	song.SetReleasedAt(releasedAt)

	if song.ReleasedAt() != releasedAt {
		t.Errorf("ReleasedAt() = %v, want %v", song.ReleasedAt(), releasedAt)
	}
}

func TestSong_Delete(t *testing.T) {
	displayID := vo.MustNewDisplayID()
	officialIdx, _ := vo.NewOfficialIdx("12345")

	song := entity.NewSong(displayID, "Test", "Test", 1, officialIdx, false)

	if song.IsDeleted() {
		t.Error("IsDeleted() = true before Delete(), want false")
	}

	song.Delete()

	if !song.IsDeleted() {
		t.Error("IsDeleted() = false after Delete(), want true")
	}
}

func TestReconstructSong(t *testing.T) {
	t.Run("正常系: DBからSongを再構築", func(t *testing.T) {
		displayID := vo.ReconstructDisplayID("abc123def456")
		officialIdx := vo.ReconstructOfficialIdx("12345")
		jacket := vo.ReconstructJacketImage("jacket123")
		bpm := 180
		releasedAt := vo.ReconstructReleaseDate("2024-01-15")

		song := entity.ReconstructSong(
			100,
			displayID,
			"Reconstructed Song",
			"Reconstructed Artist",
			3,
			officialIdx,
			jacket,
			&bpm,
			releasedAt,
			true,
			false,
		)

		if song.ID() != 100 {
			t.Errorf("ID() = %v, want 100", song.ID())
		}
		if song.DisplayID() != displayID {
			t.Errorf("DisplayID() = %v, want %v", song.DisplayID(), displayID)
		}
		if song.Jacket() != jacket {
			t.Errorf("Jacket() = %v, want %v", song.Jacket(), jacket)
		}
		if song.BPM() == nil || *song.BPM() != bpm {
			t.Errorf("BPM() = %v, want %v", song.BPM(), bpm)
		}
		if song.ReleasedAt() != releasedAt {
			t.Errorf("ReleasedAt() = %v, want %v", song.ReleasedAt(), releasedAt)
		}
		if !song.IsWorldsEnd() {
			t.Error("IsWorldsEnd() = false, want true")
		}
	})
}
