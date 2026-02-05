package entity

import (
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

// WorldsEndChart はWORLD'S END譜面を表すドメインエンティティ
type WorldsEndChart struct {
	songID  int
	weStar  vo.WeStar
	weKanji vo.WeKanji
	notes   *int
}

// NewWorldsEndChart は新しいWorldsEndChartエンティティを生成します
func NewWorldsEndChart(songID int, weStar vo.WeStar, weKanji vo.WeKanji) *WorldsEndChart {
	return &WorldsEndChart{
		songID:  songID,
		weStar:  weStar,
		weKanji: weKanji,
	}
}

// ReconstructWorldsEndChart はDBから読み込んだデータでWorldsEndChartを再構築します
func ReconstructWorldsEndChart(
	songID int,
	weStar vo.WeStar,
	weKanji vo.WeKanji,
	notes *int,
) *WorldsEndChart {
	return &WorldsEndChart{
		songID:  songID,
		weStar:  weStar,
		weKanji: weKanji,
		notes:   notes,
	}
}

// SongID は楽曲IDを返します
func (c *WorldsEndChart) SongID() int { return c.songID }

// WeStar は星数を返します
func (c *WorldsEndChart) WeStar() vo.WeStar { return c.weStar }

// WeKanji はカテゴリ漢字を返します
func (c *WorldsEndChart) WeKanji() vo.WeKanji { return c.weKanji }

// Notes はノーツ数を返します
func (c *WorldsEndChart) Notes() *int { return c.notes }

// SetNotes はノーツ数を設定します
func (c *WorldsEndChart) SetNotes(notes int) {
	c.notes = &notes
}

// SetWeStar は星数を設定します
func (c *WorldsEndChart) SetWeStar(weStar vo.WeStar) {
	c.weStar = weStar
}

// SetWeKanji はカテゴリ漢字を設定します
func (c *WorldsEndChart) SetWeKanji(weKanji vo.WeKanji) {
	c.weKanji = weKanji
}
