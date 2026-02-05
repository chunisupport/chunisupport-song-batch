package entity_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

func TestNewWorldsEndChart(t *testing.T) {
	t.Run("正常系: 新しいWorldsEndChartが生成される", func(t *testing.T) {
		weStar, _ := vo.NewWeStar(3)
		weKanji := vo.NewWeKanji("狂")

		chart := entity.NewWorldsEndChart(1, weStar, weKanji)

		if chart.SongID() != 1 {
			t.Errorf("SongID() = %v, want 1", chart.SongID())
		}
		if chart.WeStar() != weStar {
			t.Errorf("WeStar() = %v, want %v", chart.WeStar(), weStar)
		}
		if chart.WeKanji() != weKanji {
			t.Errorf("WeKanji() = %v, want %v", chart.WeKanji(), weKanji)
		}
		if chart.Notes() != nil {
			t.Errorf("Notes() = %v, want nil", chart.Notes())
		}
	})
}

func TestWorldsEndChart_SetNotes(t *testing.T) {
	weStar, _ := vo.NewWeStar(3)
	weKanji := vo.NewWeKanji("狂")

	chart := entity.NewWorldsEndChart(1, weStar, weKanji)

	chart.SetNotes(500)

	if chart.Notes() == nil {
		t.Error("Notes() = nil, want 500")
	} else if *chart.Notes() != 500 {
		t.Errorf("Notes() = %v, want 500", *chart.Notes())
	}
}

func TestWorldsEndChart_SetWeStar(t *testing.T) {
	weStar1, _ := vo.NewWeStar(3)
	weStar2, _ := vo.NewWeStar(5)
	weKanji := vo.NewWeKanji("狂")

	chart := entity.NewWorldsEndChart(1, weStar1, weKanji)

	chart.SetWeStar(weStar2)

	if chart.WeStar() != weStar2 {
		t.Errorf("WeStar() = %v, want %v", chart.WeStar(), weStar2)
	}
}

func TestWorldsEndChart_SetWeKanji(t *testing.T) {
	weStar, _ := vo.NewWeStar(3)
	weKanji1 := vo.NewWeKanji("狂")
	weKanji2 := vo.NewWeKanji("弾")

	chart := entity.NewWorldsEndChart(1, weStar, weKanji1)

	chart.SetWeKanji(weKanji2)

	if chart.WeKanji() != weKanji2 {
		t.Errorf("WeKanji() = %v, want %v", chart.WeKanji(), weKanji2)
	}
}

func TestReconstructWorldsEndChart(t *testing.T) {
	weStar := vo.ReconstructWeStar(4)
	weKanji := vo.ReconstructWeKanji("招")
	notes := 800

	chart := entity.ReconstructWorldsEndChart(
		100,
		weStar,
		weKanji,
		&notes,
	)

	if chart.SongID() != 100 {
		t.Errorf("SongID() = %v, want 100", chart.SongID())
	}
	if chart.WeStar() != weStar {
		t.Errorf("WeStar() = %v, want %v", chart.WeStar(), weStar)
	}
	if chart.WeKanji() != weKanji {
		t.Errorf("WeKanji() = %v, want %v", chart.WeKanji(), weKanji)
	}
	if chart.Notes() == nil || *chart.Notes() != notes {
		t.Errorf("Notes() = %v, want %v", chart.Notes(), notes)
	}
}
