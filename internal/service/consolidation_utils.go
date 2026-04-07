package service

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"

	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/jmoiron/sqlx"
)

type ConsolidationUtils struct {
	db *sqlx.DB
}

// ChartNotesRecord は notes の一括更新に使用される内部レコードです。
type ChartNotesRecord struct {
	SongID       int `db:"song_id"`
	DifficultyID int `db:"difficulty_id"`
	Notes        int `db:"notes"`
}

// ChartNotesDesignerRecord は notes_designer の一括更新に使用される内部レコードです。
type ChartNotesDesignerRecord struct {
	SongID        int    `db:"song_id"`
	DifficultyID  int    `db:"difficulty_id"`
	NotesDesigner string `db:"notes_designer"`
}

// bulkUpdateChartNotesTpl は notes の一括更新用テンプレートです。
// SQLiteでは UPDATE ... FROM 構文が使えないため、CASE を使います。
// 差分検知により、既存の notes を 0 や null で上書きしないようにします。
var bulkUpdateChartNotesTpl = template.Must(template.New("bulkUpdateChartNotes").Parse(`
UPDATE charts SET notes = CASE
	{{- range .}}
	WHEN song_id = {{.SongID}} AND difficulty_id = {{.DifficultyID}} THEN {{.Notes}}
	{{- end}}
	ELSE notes
END
WHERE EXISTS (
	SELECT 1 FROM (
		{{- range $i, $e := .}}
		{{- if $i}} UNION ALL{{end}}
		SELECT {{.SongID}} AS song_id, {{.DifficultyID}} AS difficulty_id, {{.Notes}} AS new_notes
		{{- end}}
	) AS t
	WHERE charts.song_id = t.song_id
	  AND charts.difficulty_id = t.difficulty_id
	  AND (charts.notes IS NULL OR charts.notes = 0)
	  AND t.new_notes > 0
)
`))

// bulkUpdateChartNotesDesignerTpl は notes_designer の一括更新用テンプレートです。
// 未設定(nullまたは空文字)のレコードにのみ譜面製作者を補完します。
var bulkUpdateChartNotesDesignerTpl = template.Must(template.New("bulkUpdateChartNotesDesigner").Funcs(template.FuncMap{
	"sqlString": escapeSQLiteStringLiteral,
}).Parse(`
UPDATE charts SET notes_designer = CASE
	{{- range .}}
	WHEN song_id = {{.SongID}} AND difficulty_id = {{.DifficultyID}} THEN '{{sqlString .NotesDesigner}}'
	{{- end}}
	ELSE notes_designer
END
WHERE EXISTS (
	SELECT 1 FROM (
		{{- range $i, $e := .}}
		{{- if $i}} UNION ALL{{end}}
		SELECT {{.SongID}} AS song_id, {{.DifficultyID}} AS difficulty_id, '{{sqlString .NotesDesigner}}' AS new_notes_designer
		{{- end}}
	) AS t
	WHERE charts.song_id = t.song_id
	  AND charts.difficulty_id = t.difficulty_id
	  AND charts.notes_designer IS NULL
	  AND t.new_notes_designer <> ''
)
`))

func escapeSQLiteStringLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
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

// ExecuteBulkUpdateChartNotes は charts テーブルの notes を1バッチ分まとめて更新します。
func ExecuteBulkUpdateChartNotes(ctx context.Context, db sqlx.ExtContext, records []ChartNotesRecord) (int64, error) {
	var buf bytes.Buffer
	if err := bulkUpdateChartNotesTpl.Execute(&buf, records); err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes template: %w", err)
	}

	result, err := db.ExecContext(ctx, buf.String())
	if err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes: %w", err)
	}

	return result.RowsAffected()
}

// BulkUpdateChartNotesInBatches は notes をバッチに分割して一括更新します。
func BulkUpdateChartNotesInBatches(ctx context.Context, db sqlx.ExtContext, records []ChartNotesRecord) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	if info.SQLiteCompoundSelectLimit <= 0 {
		return 0, fmt.Errorf("invalid SQLiteCompoundSelectLimit: %d", info.SQLiteCompoundSelectLimit)
	}

	var totalAffected int64
	for i := 0; i < len(records); i += info.SQLiteCompoundSelectLimit {
		if err := ctx.Err(); err != nil {
			return totalAffected, err
		}

		end := min(i+info.SQLiteCompoundSelectLimit, len(records))
		batch := records[i:end]

		affected, err := ExecuteBulkUpdateChartNotes(ctx, db, batch)
		if err != nil {
			return totalAffected, fmt.Errorf("failed to execute bulk update for batch range [%d:%d): %w", i, end, err)
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// ExecuteBulkUpdateChartNotesDesigner は charts テーブルの notes_designer を1バッチ分まとめて更新します。
func ExecuteBulkUpdateChartNotesDesigner(ctx context.Context, db sqlx.ExtContext, records []ChartNotesDesignerRecord) (int64, error) {
	var buf bytes.Buffer
	if err := bulkUpdateChartNotesDesignerTpl.Execute(&buf, records); err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes_designer template: %w", err)
	}

	result, err := db.ExecContext(ctx, buf.String())
	if err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes_designer: %w", err)
	}

	return result.RowsAffected()
}

// BulkUpdateChartNotesDesignerInBatches は notes_designer をバッチに分割して一括更新します。
func BulkUpdateChartNotesDesignerInBatches(ctx context.Context, db sqlx.ExtContext, records []ChartNotesDesignerRecord) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	if info.SQLiteCompoundSelectLimit <= 0 {
		return 0, fmt.Errorf("invalid SQLiteCompoundSelectLimit: %d", info.SQLiteCompoundSelectLimit)
	}

	var totalAffected int64
	for i := 0; i < len(records); i += info.SQLiteCompoundSelectLimit {
		if err := ctx.Err(); err != nil {
			return totalAffected, err
		}

		end := min(i+info.SQLiteCompoundSelectLimit, len(records))
		batch := records[i:end]

		affected, err := ExecuteBulkUpdateChartNotesDesigner(ctx, db, batch)
		if err != nil {
			return totalAffected, fmt.Errorf("failed to execute bulk update notes_designer for batch range [%d:%d): %w", i, end, err)
		}
		totalAffected += affected
	}

	return totalAffected, nil
}
