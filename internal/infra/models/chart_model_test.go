package models_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/difficulty"
	"chunisupport-song-batch/internal/domain/entity"
	vo "chunisupport-song-batch/internal/domain/valueobject"
	"chunisupport-song-batch/internal/infra/models"
)

func TestFromChartEntity(t *testing.T) {
	t.Run("正常系: Chartエンティティからモデルに変換", func(t *testing.T) {
		level, _ := vo.ParseLevel("14+")
		chart := entity.NewChart(100, difficulty.Master, level, true)
		chart.SetNotes(1500)

		model := models.FromChartEntity(chart)

		if model.SongID != 100 {
			t.Errorf("SongID = %v, want 100", model.SongID)
		}
		if model.DifficultyID != difficulty.Master.Int() {
			t.Errorf("DifficultyID = %v, want %v", model.DifficultyID, difficulty.Master.Int())
		}
		if model.Const != 14.5 {
			t.Errorf("Const = %v, want 14.5", model.Const)
		}
		if model.IsConstUnknown != 1 {
			t.Errorf("IsConstUnknown = %v, want 1", model.IsConstUnknown)
		}
		if model.Notes == nil || *model.Notes != 1500 {
			t.Errorf("Notes = %v, want 1500", model.Notes)
		}
	})
}

func TestChartModel_ToChartEntity(t *testing.T) {
	t.Run("正常系: モデルからChartエンティティに変換", func(t *testing.T) {
		notes := 2000
		model := &models.ChartModel{
			SongID:         100,
			DifficultyID:   difficulty.Ultima.Int(),
			Const:          15.0,
			IsConstUnknown: 0,
			Notes:          &notes,
		}

		chart := model.ToChartEntity()

		if chart.SongID() != 100 {
			t.Errorf("SongID() = %v, want 100", chart.SongID())
		}
		if chart.DifficultyID() != difficulty.Ultima {
			t.Errorf("DifficultyID() = %v, want %v", chart.DifficultyID(), difficulty.Ultima)
		}
		if chart.Level().Float64() != 15.0 {
			t.Errorf("Level() = %v, want 15.0", chart.Level().Float64())
		}
		if chart.IsConstUnknown() {
			t.Error("IsConstUnknown() = true, want false")
		}
		if chart.Notes() == nil || *chart.Notes() != 2000 {
			t.Errorf("Notes() = %v, want 2000", chart.Notes())
		}
	})
}

func TestFromChartEntityForUpsert(t *testing.T) {
	t.Run("正常系: UPSERT用モデルに変換", func(t *testing.T) {
		level, _ := vo.ParseLevel("13")
		chart := entity.NewChart(100, difficulty.Expert, level, false)

		model := models.FromChartEntityForUpsert(chart)

		if model.SongID != 100 {
			t.Errorf("SongID = %v, want 100", model.SongID)
		}
		if model.DifficultyID != difficulty.Expert.Int() {
			t.Errorf("DifficultyID = %v, want %v", model.DifficultyID, difficulty.Expert.Int())
		}
		if model.Const != 13.0 {
			t.Errorf("Const = %v, want 13.0", model.Const)
		}
		if model.IsConstUnknown != 0 {
			t.Errorf("IsConstUnknown = %v, want 0", model.IsConstUnknown)
		}
	})
}
