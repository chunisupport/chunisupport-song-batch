package models_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/models"
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
		if model.LevelStar == nil || *model.LevelStar != 3 {
			t.Errorf("LevelStar = %v, want 3", model.LevelStar)
		}
		if model.Attribute == nil || *model.Attribute != "狂" {
			t.Errorf("Attribute = %v, want '狂'", model.Attribute)
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

		if model.LevelStar != nil {
			t.Errorf("LevelStar = %v, want nil", model.LevelStar)
		}
		if model.Attribute != nil {
			t.Errorf("Attribute = %v, want nil", model.Attribute)
		}
	})
}

func TestWorldsEndChartModel_ToWorldsEndChartEntity(t *testing.T) {
	t.Run("正常系: モデルからWorldsEndChartエンティティに変換", func(t *testing.T) {
		levelStar := 4
		attribute := "招"
		notes := 800
		model := &models.WorldsEndChartModel{
			SongID:    100,
			LevelStar: &levelStar,
			Attribute: &attribute,
			Notes:     &notes,
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
			SongID:    100,
			LevelStar: nil,
			Attribute: nil,
			Notes:     nil,
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
