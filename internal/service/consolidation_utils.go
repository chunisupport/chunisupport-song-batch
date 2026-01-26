package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

// LookupGenreID はジャンル名からジャンルIDを検索します。
// genreIDByKey マップが空の場合は 0 を返します。
func LookupGenreID(genreIDByKey map[string]int, genre string) int {
	if len(genreIDByKey) == 0 {
		return 0
	}
	key := strings.TrimSpace(strings.ToUpper(genre))
	return genreIDByKey[key]
}

// songIDRecord は official_idx から song_id を取得する際の内部型です
type songIDRecord struct {
	ID          int    `db:"id"`
	OfficialIdx string `db:"official_idx"`
}

// FetchSongIDsByOfficialIdx は指定された official_idx のセットに対応する song_id のマップを返します。
// 主にバルク処理でUPSERT後のIDを取得する際に使用します。
func FetchSongIDsByOfficialIdx(ctx context.Context, db sqlx.QueryerContext, officialIdxs map[string]struct{}) (map[string]int, error) {
	if len(officialIdxs) == 0 {
		return make(map[string]int), nil
	}

	idxList := make([]string, 0, len(officialIdxs))
	for idx := range officialIdxs {
		idxList = append(idxList, idx)
	}

	query, args, err := sqlx.In(`SELECT id, official_idx FROM songs WHERE official_idx IN (?)`, idxList)
	if err != nil {
		return nil, fmt.Errorf("failed to build query for fetching song IDs: %w", err)
	}

	var records []songIDRecord
	if err := sqlx.SelectContext(ctx, db, &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch song IDs by official_idx: %w", err)
	}

	result := make(map[string]int, len(records))
	for _, r := range records {
		result[r.OfficialIdx] = r.ID
	}
	return result, nil
}

// BuildOfficialIndexMap は全ての official_idx と song_id のマップを返します。
// natua等で全件取得が必要な場合に使用します。
func BuildOfficialIndexMap(ctx context.Context, db sqlx.QueryerContext) (map[string]int, error) {
	var rows []songIDRecord
	if err := sqlx.SelectContext(ctx, db, &rows, `SELECT id, official_idx FROM songs WHERE official_idx IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("failed to build workspace official_idx map: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.OfficialIdx] = row.ID
	}
	return result, nil
}
