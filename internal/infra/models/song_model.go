package models

import (
	"chunisupport-song-batch/internal/domain/entity"
	vo "chunisupport-song-batch/internal/domain/valueobject"
	"chunisupport-song-batch/internal/util"
)

// SongModel はDB永続化用の楽曲構造体
type SongModel struct {
	ID          int     `db:"id"`
	DisplayID   string  `db:"display_id"`
	Title       string  `db:"title"`
	Artist      string  `db:"artist"`
	GenreID     int     `db:"genre_id"`
	OfficialIdx string  `db:"official_idx"`
	Jacket      any     `db:"jacket"`
	BPM         *int    `db:"bpm"`
	ReleasedAt  *string `db:"released_at"`
	IsWorldsEnd int     `db:"is_worldsend"`
	IsDeleted   int     `db:"is_deleted"`
}

// FromSongEntity はSongエンティティからSongModelを生成します
func FromSongEntity(s *entity.Song) *SongModel {
	return &SongModel{
		ID:          s.ID(),
		DisplayID:   s.DisplayID().String(),
		Title:       s.Title(),
		Artist:      s.Artist(),
		GenreID:     s.GenreID(),
		OfficialIdx: s.OfficialIdx().String(),
		Jacket:      s.Jacket().NullableString(),
		BPM:         s.BPM(),
		ReleasedAt:  s.ReleasedAt().StringPtr(),
		IsWorldsEnd: util.BoolToInt(s.IsWorldsEnd()),
		IsDeleted:   util.BoolToInt(s.IsDeleted()),
	}
}

// ToSongEntity はSongModelからSongエンティティを生成します
func (m *SongModel) ToSongEntity() *entity.Song {
	var releasedAt vo.ReleaseDate
	if m.ReleasedAt != nil {
		releasedAt, _ = vo.ParseReleaseDate(*m.ReleasedAt)
	}

	var jacket vo.JacketImage
	if m.Jacket != nil {
		if s, ok := m.Jacket.(string); ok {
			jacket = vo.ReconstructJacketImage(s)
		}
	}

	officialIdx := vo.ReconstructOfficialIdx(m.OfficialIdx)

	return entity.ReconstructSong(
		m.ID,
		vo.ReconstructDisplayID(m.DisplayID),
		m.Title,
		m.Artist,
		m.GenreID,
		officialIdx,
		jacket,
		m.BPM,
		releasedAt,
		m.IsWorldsEnd == 1,
		m.IsDeleted == 1,
	)
}

// SongModelForUpsert はUPSERT操作用の楽曲モデルです
type SongModelForUpsert struct {
	DisplayID   string  `db:"display_id"`
	Title       string  `db:"title"`
	Artist      string  `db:"artist"`
	GenreID     int     `db:"genre_id"`
	OfficialIdx string  `db:"official_idx"`
	Jacket      any     `db:"jacket"`
	BPM         *int    `db:"bpm"`
	ReleasedAt  *string `db:"released_at"`
	IsWorldsEnd int     `db:"is_worldsend"`
}

// FromSongEntityForUpsert はSongエンティティからUPSERT用モデルを生成します
func FromSongEntityForUpsert(s *entity.Song) *SongModelForUpsert {
	return &SongModelForUpsert{
		DisplayID:   s.DisplayID().String(),
		Title:       s.Title(),
		Artist:      s.Artist(),
		GenreID:     s.GenreID(),
		OfficialIdx: s.OfficialIdx().String(),
		Jacket:      s.Jacket().NullableString(),
		BPM:         s.BPM(),
		ReleasedAt:  s.ReleasedAt().StringPtr(),
		IsWorldsEnd: util.BoolToInt(s.IsWorldsEnd()),
	}
}
