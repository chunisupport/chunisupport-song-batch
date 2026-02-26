package models

import (
	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

// WorldsEndChartModel はDB永続化用のWORLD'S END譜面構造体
type WorldsEndChartModel struct {
	SongID    int     `db:"song_id"`
	LevelStar *int    `db:"level_star"`
	Attribute *string `db:"attribute"`
	Notes     *int    `db:"notes"`
}

// FromWorldsEndChartEntity はWorldsEndChartエンティティからWorldsEndChartModelを生成します
func FromWorldsEndChartEntity(c *entity.WorldsEndChart) *WorldsEndChartModel {
	return &WorldsEndChartModel{
		SongID:    c.SongID(),
		LevelStar: c.WeStar().IntPtr(),
		Attribute: c.WeKanji().StringPtr(),
		Notes:     c.Notes(),
	}
}

// ToWorldsEndChartEntity はWorldsEndChartModelからWorldsEndChartエンティティを生成します
func (m *WorldsEndChartModel) ToWorldsEndChartEntity() *entity.WorldsEndChart {
	var weStar vo.WeStar
	if m.LevelStar != nil {
		weStar = vo.ReconstructWeStar(*m.LevelStar)
	}

	var weKanji vo.WeKanji
	if m.Attribute != nil {
		weKanji = vo.ReconstructWeKanji(*m.Attribute)
	}

	return entity.ReconstructWorldsEndChart(
		m.SongID,
		weStar,
		weKanji,
		m.Notes,
	)
}

// WorldsEndChartModelForUpsert はUPSERT操作用のWORLD'S END譜面モデルです
type WorldsEndChartModelForUpsert struct {
	SongID    int     `db:"song_id"`
	LevelStar *int    `db:"level_star"`
	Attribute *string `db:"attribute"`
}

// FromWorldsEndChartEntityForUpsert はWorldsEndChartエンティティからUPSERT用モデルを生成します
func FromWorldsEndChartEntityForUpsert(c *entity.WorldsEndChart) *WorldsEndChartModelForUpsert {
	return &WorldsEndChartModelForUpsert{
		SongID:    c.SongID(),
		LevelStar: c.WeStar().IntPtr(),
		Attribute: c.WeKanji().StringPtr(),
	}
}
