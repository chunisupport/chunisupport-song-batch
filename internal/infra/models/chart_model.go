package models

import (
	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
	"github.com/chunisupport/chunisupport-song-batch/internal/util"
)

// ChartModel はDB永続化用の譜面構造体
type ChartModel struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
	Notes          *int    `db:"notes"`
}

// FromChartEntity はChartエンティティからChartModelを生成します
func FromChartEntity(c *entity.Chart) *ChartModel {
	return &ChartModel{
		SongID:         c.SongID(),
		DifficultyID:   c.DifficultyID().Int(),
		Const:          c.Level().Float64(),
		IsConstUnknown: util.BoolToInt(c.IsConstUnknown()),
		Notes:          c.Notes(),
	}
}

// ToChartEntity はChartModelからChartエンティティを生成します
func (m *ChartModel) ToChartEntity() *entity.Chart {
	return entity.ReconstructChart(
		m.SongID,
		difficulty.ID(m.DifficultyID),
		vo.NewLevel(m.Const),
		m.IsConstUnknown == 1,
		m.Notes,
	)
}

// ChartModelForUpsert はUPSERT操作用の譜面モデルです
type ChartModelForUpsert struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
}

// FromChartEntityForUpsert はChartエンティティからUPSERT用モデルを生成します
func FromChartEntityForUpsert(c *entity.Chart) *ChartModelForUpsert {
	return &ChartModelForUpsert{
		SongID:         c.SongID(),
		DifficultyID:   c.DifficultyID().Int(),
		Const:          c.Level().Float64(),
		IsConstUnknown: util.BoolToInt(c.IsConstUnknown()),
	}
}

// ChartModelForUpsertWithNotes はノーツ付きUPSERT操作用の譜面モデルです
type ChartModelForUpsertWithNotes struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
	Notes          *int    `db:"notes"`
}

// FromChartEntityForUpsertWithNotes はChartエンティティからノーツ付きUPSERT用モデルを生成します
func FromChartEntityForUpsertWithNotes(c *entity.Chart) *ChartModelForUpsertWithNotes {
	return &ChartModelForUpsertWithNotes{
		SongID:         c.SongID(),
		DifficultyID:   c.DifficultyID().Int(),
		Const:          c.Level().Float64(),
		IsConstUnknown: util.BoolToInt(c.IsConstUnknown()),
		Notes:          c.Notes(),
	}
}
