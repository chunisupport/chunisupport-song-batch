package entity

import (
	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

// Song は楽曲を表すドメインエンティティ
type Song struct {
	id          int
	displayID   vo.DisplayID
	title       string
	artist      string
	genreID     int
	officialIdx vo.OfficialIdx
	jacket      vo.JacketImage
	bpm         *int
	releasedAt  vo.ReleaseDate
	isWorldsEnd bool
	isDeleted   bool
}

// NewSong は新しいSongエンティティを生成します
func NewSong(
	displayID vo.DisplayID,
	title, artist string,
	genreID int,
	officialIdx vo.OfficialIdx,
	isWorldsEnd bool,
) *Song {
	return &Song{
		displayID:   displayID,
		title:       title,
		artist:      artist,
		genreID:     genreID,
		officialIdx: officialIdx,
		isWorldsEnd: isWorldsEnd,
		isDeleted:   false,
	}
}

// ReconstructSong はDBから読み込んだデータでSongを再構築します
func ReconstructSong(
	id int,
	displayID vo.DisplayID,
	title, artist string,
	genreID int,
	officialIdx vo.OfficialIdx,
	jacket vo.JacketImage,
	bpm *int,
	releasedAt vo.ReleaseDate,
	isWorldsEnd, isDeleted bool,
) *Song {
	return &Song{
		id:          id,
		displayID:   displayID,
		title:       title,
		artist:      artist,
		genreID:     genreID,
		officialIdx: officialIdx,
		jacket:      jacket,
		bpm:         bpm,
		releasedAt:  releasedAt,
		isWorldsEnd: isWorldsEnd,
		isDeleted:   isDeleted,
	}
}

// ID は楽曲IDを返します
func (s *Song) ID() int { return s.id }

// DisplayID は表示用IDを返します
func (s *Song) DisplayID() vo.DisplayID { return s.displayID }

// Title はタイトルを返します
func (s *Song) Title() string { return s.title }

// Artist はアーティストを返します
func (s *Song) Artist() string { return s.artist }

// GenreID はジャンルIDを返します
func (s *Song) GenreID() int { return s.genreID }

// OfficialIdx は公式IDを返します
func (s *Song) OfficialIdx() vo.OfficialIdx { return s.officialIdx }

// Jacket はジャケット画像を返します
func (s *Song) Jacket() vo.JacketImage { return s.jacket }

// BPM はBPMを返します
func (s *Song) BPM() *int { return s.bpm }

// ReleasedAt はリリース日を返します
func (s *Song) ReleasedAt() vo.ReleaseDate { return s.releasedAt }

// IsWorldsEnd はWORLD'S END楽曲かどうかを返します
func (s *Song) IsWorldsEnd() bool { return s.isWorldsEnd }

// IsDeleted は削除済みかどうかを返します
func (s *Song) IsDeleted() bool { return s.isDeleted }

// SetJacket はジャケット画像を設定します
func (s *Song) SetJacket(jacket vo.JacketImage) {
	s.jacket = jacket
}

// SetBPM はBPMを設定します
func (s *Song) SetBPM(bpm int) {
	s.bpm = &bpm
}

// SetReleasedAt はリリース日を設定します
func (s *Song) SetReleasedAt(releasedAt vo.ReleaseDate) {
	s.releasedAt = releasedAt
}

// Delete は楽曲を削除済みにします
func (s *Song) Delete() {
	s.isDeleted = true
}

// SetID はDBでの採番後にIDを設定します
// 注意: このメソッドはインフラ層からのみ使用されるべきです
func (s *Song) SetID(id int) {
	s.id = id
}
