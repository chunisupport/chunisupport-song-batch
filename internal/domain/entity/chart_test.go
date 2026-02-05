package entity_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewChart(t *testing.T) {
	t.Run("正常系: 新しいChartが生成される", func(t *testing.T) {
		level, _ := vo.ParseLevel("14+")
		chart := entity.NewChart(1, difficulty.Master, level, true)

		if chart.SongID() != 1 {
			t.Errorf("SongID() = %v, want 1", chart.SongID())
		}
		if chart.DifficultyID() != difficulty.Master {
			t.Errorf("DifficultyID() = %v, want %v", chart.DifficultyID(), difficulty.Master)
		}
		if chart.Level() != level {
			t.Errorf("Level() = %v, want %v", chart.Level(), level)
		}
		if !chart.IsConstUnknown() {
			t.Error("IsConstUnknown() = false, want true")
		}
		if chart.Notes() != nil {
			t.Errorf("Notes() = %v, want nil", chart.Notes())
		}
	})
}

func TestChart_SetLevel(t *testing.T) {
	level1, _ := vo.ParseLevel("13")
	level2, _ := vo.ParseLevel("13.5")

	chart := entity.NewChart(1, difficulty.Expert, level1, true)

	chart.SetLevel(level2)

	if chart.Level() != level2 {
		t.Errorf("Level() = %v, want %v", chart.Level(), level2)
	}
	if chart.IsConstUnknown() {
		t.Error("IsConstUnknown() = true after SetLevel(), want false")
	}
}

func TestChart_SetLevelWithUnknownFlag(t *testing.T) {
	level1, _ := vo.ParseLevel("13")
	level2, _ := vo.ParseLevel("13.5")

	chart := entity.NewChart(1, difficulty.Expert, level1, false)

	chart.SetLevelWithUnknownFlag(level2, true)

	if chart.Level() != level2 {
		t.Errorf("Level() = %v, want %v", chart.Level(), level2)
	}
	if !chart.IsConstUnknown() {
		t.Error("IsConstUnknown() = false, want true")
	}
}

func TestChart_SetNotes(t *testing.T) {
	level, _ := vo.ParseLevel("13")
	chart := entity.NewChart(1, difficulty.Expert, level, false)

	chart.SetNotes(1500)

	if chart.Notes() == nil {
		t.Error("Notes() = nil, want 1500")
	} else if *chart.Notes() != 1500 {
		t.Errorf("Notes() = %v, want 1500", *chart.Notes())
	}
}

func TestChart_MarkConstUnknown(t *testing.T) {
	level, _ := vo.ParseLevel("13")
	chart := entity.NewChart(1, difficulty.Expert, level, false)

	if chart.IsConstUnknown() {
		t.Error("IsConstUnknown() = true before MarkConstUnknown(), want false")
	}

	chart.MarkConstUnknown()

	if !chart.IsConstUnknown() {
		t.Error("IsConstUnknown() = false after MarkConstUnknown(), want true")
	}
}

func TestReconstructChart(t *testing.T) {
	level, _ := vo.ParseLevel("14.5")
	notes := 2000

	chart := entity.ReconstructChart(
		100,
		difficulty.Ultima,
		level,
		false,
		&notes,
	)

	if chart.SongID() != 100 {
		t.Errorf("SongID() = %v, want 100", chart.SongID())
	}
	if chart.DifficultyID() != difficulty.Ultima {
		t.Errorf("DifficultyID() = %v, want %v", chart.DifficultyID(), difficulty.Ultima)
	}
	if chart.Level() != level {
		t.Errorf("Level() = %v, want %v", chart.Level(), level)
	}
	if chart.IsConstUnknown() {
		t.Error("IsConstUnknown() = true, want false")
	}
	if chart.Notes() == nil || *chart.Notes() != notes {
		t.Errorf("Notes() = %v, want %v", chart.Notes(), notes)
	}
}
