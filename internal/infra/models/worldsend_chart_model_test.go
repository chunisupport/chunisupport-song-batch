package models_test

import (
	"testing"

	"chunisupport-song-batch/internal/domain/entity"
	vo "chunisupport-song-batch/internal/domain/valueobject"
	"chunisupport-song-batch/internal/infra/models"
)

func TestFromWorldsEndChartEntity(t *testing.T) {
	t.Run("正常系: WorldsEndChartエンティティからモデルに変換", func(t *testing.T) {
		weStar, _ := vo.NewWeStar(3)
		weKanji := vo.NewWeKanji("狂")
		chart := entity.NewWorldsEndChart(100, weStar, weKanji)
		chart.SetNotes(500)

		model := models.FromWorldsEndChartEntity(chart)

		if model.SongID != 100 {
			t.Errorf("SongID = %v, want 100", model.SongID)
		}
		if model.WeStar == nil || *model.WeStar != 3 {
			t.Errorf("WeStar = %v, want 3", model.WeStar)
		}
		if model.WeKanji == nil || *model.WeKanji != "狂" {
			t.Errorf("WeKanji = %v, want '狂'", model.WeKanji)
		}
		if model.Notes == nil || *model.Notes != 500 {
			t.Errorf("Notes = %v, want 500", model.Notes)
		}
	})

	t.Run("正常系: 値がゼロの場合はnil", func(t *testing.T) {
		weStar := vo.WeStar(0)
		weKanji := vo.WeKanji("")
		chart := entity.NewWorldsEndChart(100, weStar, weKanji)

		model := models.FromWorldsEndChartEntity(chart)

		if model.WeStar != nil {
			t.Errorf("WeStar = %v, want nil", model.WeStar)
		}
		if model.WeKanji != nil {
			t.Errorf("WeKanji = %v, want nil", model.WeKanji)
		}
	})
}

func TestWorldsEndChartModel_ToWorldsEndChartEntity(t *testing.T) {
	t.Run("正常系: モデルからWorldsEndChartエンティティに変換", func(t *testing.T) {
		weStar := 4
		weKanji := "招"
		notes := 800
		model := &models.WorldsEndChartModel{
			SongID:  100,
			WeStar:  &weStar,
			WeKanji: &weKanji,
			Notes:   &notes,
		}

		chart := model.ToWorldsEndChartEntity()

		if chart.SongID() != 100 {
			t.Errorf("SongID() = %v, want 100", chart.SongID())
		}
		if chart.WeStar().Int() != 4 {
			t.Errorf("WeStar() = %v, want 4", chart.WeStar().Int())
		}
		if chart.WeKanji().String() != "招" {
			t.Errorf("WeKanji() = %v, want '招'", chart.WeKanji())
		}
		if chart.Notes() == nil || *chart.Notes() != 800 {
			t.Errorf("Notes() = %v, want 800", chart.Notes())
		}
	})

	t.Run("正常系: nilの場合はゼロ値", func(t *testing.T) {
		model := &models.WorldsEndChartModel{
			SongID:  100,
			WeStar:  nil,
			WeKanji: nil,
			Notes:   nil,
		}

		chart := model.ToWorldsEndChartEntity()

		if chart.WeStar().Int() != 0 {
			t.Errorf("WeStar() = %v, want 0", chart.WeStar().Int())
		}
		if !chart.WeKanji().IsEmpty() {
			t.Errorf("WeKanji() = %v, want empty", chart.WeKanji())
		}
	})
}
