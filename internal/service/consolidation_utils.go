package service

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ConsolidationUtils struct {
	db *sqlx.DB
}

// NewConsolidationUtils は集約処理向けのユーティリティを初期化します。
func NewConsolidationUtils(db *sqlx.DB) *ConsolidationUtils {
	return &ConsolidationUtils{db: db}
}

// FindSongByOfficialID は official_idx で楽曲IDを検索します。
func (u *ConsolidationUtils) FindSongByOfficialID(officialID string) (int, error) {
	var songID int
	err := u.db.Get(&songID, `
		SELECT id FROM songs WHERE official_idx = ?
	`, officialID)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return songID, err
}

// FindSongByTitleAndArtist はタイトルとアーティストから楽曲IDを検索します。
func (u *ConsolidationUtils) FindSongByTitleAndArtist(title, artist string) (int, error) {
	var songID int
	err := u.db.Get(&songID, `
		SELECT id FROM songs WHERE title = ? AND artist = ?
	`, title, artist)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return songID, err
}

// UpsertSong は楽曲情報をUPSERTします。
func (u *ConsolidationUtils) UpsertSong(song SongData) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

// UpsertChart は譜面情報をUPSERTします。
func (u *ConsolidationUtils) UpsertChart(chart ChartData) error {
	return fmt.Errorf("not implemented")
}

type SongData struct {
	ID          int
	Title       string
	Artist      string
	GenreID     *int8
	BPM         *int
	ReleaseDate *string
	OfficialIdx *string
	ImageURL    *string
}

type ChartData struct {
	SongID         int
	DifficultyID   int8
	Level          *float32
	Const          *float32
	IsConstUnknown bool
	Notes          *int
}
