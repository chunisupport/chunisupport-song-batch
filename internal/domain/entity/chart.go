package entity

import (
	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

// Chart は譜面を表すドメインエンティティ
type Chart struct {
	songID         int
	difficultyID   difficulty.ID
	level          vo.Level
	isConstUnknown bool
	notes          *int
}

// NewChart は新しいChartエンティティを生成します
func NewChart(songID int, difficultyID difficulty.ID, level vo.Level, isConstUnknown bool) *Chart {
	return &Chart{
		songID:         songID,
		difficultyID:   difficultyID,
		level:          level,
		isConstUnknown: isConstUnknown,
	}
}

// ReconstructChart はDBから読み込んだデータでChartを再構築します
func ReconstructChart(
	songID int,
	difficultyID difficulty.ID,
	level vo.Level,
	isConstUnknown bool,
	notes *int,
) *Chart {
	return &Chart{
		songID:         songID,
		difficultyID:   difficultyID,
		level:          level,
		isConstUnknown: isConstUnknown,
		notes:          notes,
	}
}

// SongID は楽曲IDを返します
func (c *Chart) SongID() int { return c.songID }

// DifficultyID は難易度IDを返します
func (c *Chart) DifficultyID() difficulty.ID { return c.difficultyID }

// Level はレベル（定数）を返します
func (c *Chart) Level() vo.Level { return c.level }

// IsConstUnknown は定数が不明かどうかを返します
func (c *Chart) IsConstUnknown() bool { return c.isConstUnknown }

// Notes はノーツ数を返します
func (c *Chart) Notes() *int { return c.notes }

// SetLevel はレベルを設定し、定数不明フラグをクリアします
func (c *Chart) SetLevel(level vo.Level) {
	c.level = level
	c.isConstUnknown = false
}

// SetLevelWithUnknownFlag はレベルと定数不明フラグを設定します
func (c *Chart) SetLevelWithUnknownFlag(level vo.Level, isConstUnknown bool) {
	c.level = level
	c.isConstUnknown = isConstUnknown
}

// SetNotes はノーツ数を設定します
func (c *Chart) SetNotes(notes int) {
	c.notes = &notes
}

// MarkConstUnknown は定数を不明としてマークします
func (c *Chart) MarkConstUnknown() {
	c.isConstUnknown = true
}
