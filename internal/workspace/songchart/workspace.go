package songchart

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/chunisupport/chunisupport-song-batch/internal/util"

	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite"
)

const (
	defaultWorkspaceDSN = "file:chuni_ws?mode=memory&cache=shared&_pragma=foreign_keys(ON)"
)

//go:embed schema.sql
var workspaceSchema string

// Config はワークスペース生成時のオプションです。
type Config struct {
	DSN string
}

// SyncOptions は MySQL への同期挙動を制御します。
type SyncOptions struct {
	MajorUpdate bool
}

// SongChartWorkspace は songs/charts を扱う SQLite ワークスペースを表します。
type SongChartWorkspace struct {
	db *sqlx.DB
}

// NewSongChartWorkspace は SQLite ワークスペースを構築します。
func NewSongChartWorkspace(ctx context.Context, cfg Config) (*SongChartWorkspace, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = defaultWorkspaceDSN
	}

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite workspace: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite workspace: %w", err)
	}

	ws := &SongChartWorkspace{db: db}
	if err := ws.bootstrapSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return ws, nil
}

// Close はワークスペース接続を閉じます。
func (w *SongChartWorkspace) Close() error {
	if w == nil || w.db == nil {
		return nil
	}
	return w.db.Close()
}

// DB は内部の *sqlx.DB を返します。
func (w *SongChartWorkspace) DB() *sqlx.DB {
	return w.db
}

func (w *SongChartWorkspace) bootstrapSchema(ctx context.Context) error {
	for _, stmt := range parseSQLStatements(workspaceSchema) {
		if _, err := w.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to initialize workspace schema: %w", err)
		}
	}

	return nil
}

// SyncToMySQL はワークスペース内容を MySQL songs/charts に反映します。
func (w *SongChartWorkspace) SyncToMySQL(ctx context.Context, mysql domainrepo.DBExecutor, opts SyncOptions) error {
	slog.Info("Starting workspace synchronization to MySQL")

	wsSongs, err := w.loadWorkspaceSongs(ctx)
	if err != nil {
		return err
	}
	wsCharts, err := w.loadWorkspaceCharts(ctx)
	if err != nil {
		return err
	}
	wsWorldsendCharts, err := w.loadWorkspaceWorldsendCharts(ctx)
	if err != nil {
		return err
	}

	mysqlSongs, err := loadMySQLSongs(ctx, mysql)
	if err != nil {
		return err
	}
	mysqlCharts, err := loadMySQLCharts(ctx, mysql)
	if err != nil {
		return err
	}
	mysqlWorldsendCharts, err := loadMySQLWorldsendCharts(ctx, mysql)
	if err != nil {
		return err
	}

	officialSeen := make(map[string]struct{}, len(wsSongs))
	songInsertRecords := make([]songInsertRecord, 0, len(wsSongs))

	for _, song := range wsSongs {
		if song.OfficialIdx == "" {
			slog.Warn("Skipping workspace song without official_idx", "song_id", song.ID)
			continue
		}

		officialSeen[song.OfficialIdx] = struct{}{}
		songInsertRecords = append(songInsertRecords, songInsertRecord{
			DisplayID:   song.DisplayID,
			Title:       song.Title,
			Artist:      song.Artist,
			GenreID:     song.GenreID,
			BPM:         song.BPM,
			ReleasedAt:  song.ReleasedAt,
			OfficialIdx: song.OfficialIdx,
			Jacket:      song.Jacket,
			IsWorldsend: song.IsWorldsend,
		})
	}

	if err := bulkUpsertMySQLSongs(ctx, mysql, songInsertRecords, info.BulkInsertChunkSize); err != nil {
		return err
	}

	mysqlSongs, err = loadMySQLSongs(ctx, mysql)
	if err != nil {
		return err
	}

	songIDMap := make(map[int]int, len(wsSongs))
	for _, song := range wsSongs {
		if song.OfficialIdx == "" {
			continue
		}
		mysqlSong, ok := mysqlSongs[song.OfficialIdx]
		if !ok {
			return fmt.Errorf("failed to resolve mysql song id for official_idx %s", song.OfficialIdx)
		}
		songIDMap[song.ID] = mysqlSong.ID
	}

	var chartsToInsert []chartInsertRecord
	var chartsToUpdate []chartUpdateRecord

	for _, chart := range wsCharts {
		mysqlSongID, ok := songIDMap[chart.SongID]
		if !ok {
			return fmt.Errorf("workspace chart references unknown song_id %d", chart.SongID)
		}

		key := chartKey(mysqlSongID, chart.DifficultyID)
		existing, exists := mysqlCharts[key]

		finalConst, finalUnknown, finalNotes, action := resolveChartUpdate(existing, exists, chart, opts)

		switch action {
		case actionInsert:
			chartsToInsert = append(chartsToInsert, chartInsertRecord{
				SongID:        mysqlSongID,
				DifficultyID:  chart.DifficultyID,
				Const:         finalConst,
				TargetUnknown: finalUnknown,
				Notes:         finalNotes,
			})
		case actionUpdate:
			chartsToUpdate = append(chartsToUpdate, chartUpdateRecord{
				SongID:        mysqlSongID,
				DifficultyID:  chart.DifficultyID,
				Const:         finalConst,
				TargetUnknown: finalUnknown,
				Notes:         finalNotes,
			})
		}
	}

	if err := bulkInsertMySQLCharts(ctx, mysql, chartsToInsert, info.BulkInsertChunkSize); err != nil {
		return err
	}
	if err := bulkUpdateMySQLCharts(ctx, mysql, chartsToUpdate, info.BulkInsertChunkSize); err != nil {
		return err
	}

	// WORLD'S END charts の同期
	var worldsendChartsToInsert []worldsendChartInsertRecord
	var worldsendChartsToUpdate []worldsendChartUpdateRecord

	for _, weChart := range wsWorldsendCharts {
		mysqlSongID, ok := songIDMap[weChart.SongID]
		if !ok {
			return fmt.Errorf("workspace worldsend_chart references unknown song_id %d", weChart.SongID)
		}

		_, exists := mysqlWorldsendCharts[mysqlSongID]
		if !exists {
			worldsendChartsToInsert = append(worldsendChartsToInsert, worldsendChartInsertRecord{
				SongID:    mysqlSongID,
				LevelStar: weChart.LevelStar,
				Attribute: weChart.Attribute,
				Notes:     weChart.Notes,
			})
		} else {
			worldsendChartsToUpdate = append(worldsendChartsToUpdate, worldsendChartUpdateRecord{
				SongID:    mysqlSongID,
				LevelStar: weChart.LevelStar,
				Attribute: weChart.Attribute,
				Notes:     weChart.Notes,
			})
		}
	}

	if err := bulkInsertMySQLWorldsendCharts(ctx, mysql, worldsendChartsToInsert, info.BulkInsertChunkSize); err != nil {
		return err
	}
	if err := bulkUpdateMySQLWorldsendCharts(ctx, mysql, worldsendChartsToUpdate, info.BulkInsertChunkSize); err != nil {
		return err
	}

	slog.Info("Workspace synchronization to MySQL completed",
		"songs", len(wsSongs),
		"charts", len(wsCharts),
		"worldsend_charts", len(wsWorldsendCharts))
	return nil
}

type workspaceSong struct {
	ID          int            `db:"id"`
	DisplayID   string         `db:"display_id"`
	Title       string         `db:"title"`
	Artist      string         `db:"artist"`
	GenreID     sql.NullInt64  `db:"genre_id"`
	BPM         sql.NullInt64  `db:"bpm"`
	ReleasedAt  sql.NullString `db:"released_at"`
	OfficialIdx string         `db:"official_idx"`
	Jacket      sql.NullString `db:"jacket"`
	IsWorldsend int            `db:"is_worldsend"`
	IsDeleted   int            `db:"is_deleted"`
}

type workspaceChart struct {
	ID             int           `db:"id"`
	SongID         int           `db:"song_id"`
	DifficultyID   int           `db:"difficulty_id"`
	Const          float64       `db:"const"`
	IsConstUnknown bool          `db:"is_const_unknown"`
	Notes          sql.NullInt64 `db:"notes"`
}

type workspaceWorldsendChart struct {
	ID        int            `db:"id"`
	SongID    int            `db:"song_id"`
	LevelStar sql.NullInt64  `db:"level_star"`
	Attribute sql.NullString `db:"attribute"`
	Notes     sql.NullInt64  `db:"notes"`
}

type mysqlSong struct {
	ID          int
	OfficialIdx string
}

type mysqlChart struct {
	SongID         int
	DifficultyID   int
	Const          float64
	IsConstUnknown bool
	Notes          sql.NullInt64
}

type mysqlWorldsendChart struct {
	SongID    int
	LevelStar sql.NullInt64
	Attribute sql.NullString
	Notes     sql.NullInt64
}

func (w *SongChartWorkspace) loadWorkspaceSongs(ctx context.Context) ([]workspaceSong, error) {
	var songs []workspaceSong
	if err := w.db.SelectContext(ctx, &songs, `SELECT * FROM songs ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace songs: %w", err)
	}

	return songs, nil
}

func (w *SongChartWorkspace) loadWorkspaceCharts(ctx context.Context) ([]workspaceChart, error) {
	var charts []workspaceChart
	if err := w.db.SelectContext(ctx, &charts, `SELECT * FROM charts ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace charts: %w", err)
	}

	return charts, nil
}

func (w *SongChartWorkspace) loadWorkspaceWorldsendCharts(ctx context.Context) ([]workspaceWorldsendChart, error) {
	var charts []workspaceWorldsendChart
	if err := w.db.SelectContext(ctx, &charts, `SELECT * FROM worldsend_charts ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace worldsend_charts: %w", err)
	}

	return charts, nil
}

func loadMySQLSongs(ctx context.Context, mysql domainrepo.DBExecutor) (map[string]mysqlSong, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT id, official_idx FROM songs WHERE official_idx IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql songs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]mysqlSong)
	for rows.Next() {
		var rec mysqlSong
		if err := rows.Scan(&rec.ID, &rec.OfficialIdx); err != nil {
			return nil, fmt.Errorf("failed to scan mysql song: %w", err)
		}
		result[rec.OfficialIdx] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql songs iteration error: %w", err)
	}
	return result, nil
}

func loadMySQLCharts(ctx context.Context, mysql domainrepo.DBExecutor) (map[string]mysqlChart, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT song_id, difficulty_id, const, is_const_unknown, notes FROM charts`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql charts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]mysqlChart)
	for rows.Next() {
		var rec mysqlChart
		if err := rows.Scan(&rec.SongID, &rec.DifficultyID, &rec.Const, &rec.IsConstUnknown, &rec.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan mysql chart: %w", err)
		}
		result[chartKey(rec.SongID, rec.DifficultyID)] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql charts iteration error: %w", err)
	}
	return result, nil
}

func loadMySQLWorldsendCharts(ctx context.Context, mysql domainrepo.DBExecutor) (map[int]mysqlWorldsendChart, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT song_id, level_star, attribute, notes FROM worldsend_charts`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql worldsend_charts: %w", err)
	}
	defer rows.Close()

	result := make(map[int]mysqlWorldsendChart)
	for rows.Next() {
		var rec mysqlWorldsendChart
		if err := rows.Scan(&rec.SongID, &rec.LevelStar, &rec.Attribute, &rec.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan mysql worldsend_chart: %w", err)
		}
		result[rec.SongID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql worldsend_charts iteration error: %w", err)
	}
	return result, nil
}

type chartUpdateRecord struct {
	SongID        int
	DifficultyID  int
	Const         float64
	TargetUnknown bool
	Notes         sql.NullInt64
}

// bulkUpdateMySQLCharts 用のテンプレート（パフォーマンスのため事前にパースしておく）
var bulkUpdateChartsTpl = template.Must(template.New("bulkUpdateCharts").Parse(`
UPDATE charts
SET
	const = CASE
		{{- range .Records }}
		WHEN song_id = ? AND difficulty_id = ? THEN ?
		{{- end }}
		ELSE const
	END,
	is_const_unknown = CASE
		{{- range .Records }}
		WHEN song_id = ? AND difficulty_id = ? THEN ?
		{{- end }}
		ELSE is_const_unknown
	END,
	notes = CASE
		{{- range .Records }}
		WHEN song_id = ? AND difficulty_id = ? THEN ?
		{{- end }}
		ELSE notes
	END
WHERE (song_id, difficulty_id) IN ({{ .Placeholders }})
`))

func bulkUpdateMySQLCharts(ctx context.Context, mysql domainrepo.DBExecutor, records []chartUpdateRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		placeholders := strings.Repeat("(?,?),", len(chunk))
		placeholders = placeholders[:len(placeholders)-1]

		var buf bytes.Buffer
		tplData := map[string]any{
			"Records":      chunk,
			"Placeholders": placeholders,
		}
		if err := bulkUpdateChartsTpl.Execute(&buf, tplData); err != nil {
			return fmt.Errorf("failed to render bulk update charts template (%d-%d): %w", start, end, err)
		}

		var args []any
		for _, rec := range chunk { // const
			args = append(args, rec.SongID, rec.DifficultyID, rec.Const)
		}
		for _, rec := range chunk { // is_const_unknown
			args = append(args, rec.SongID, rec.DifficultyID, util.BoolToInt(rec.TargetUnknown))
		}
		for _, rec := range chunk { // notes
			args = append(args, rec.SongID, rec.DifficultyID, nullableInt(rec.Notes))
		}
		for _, rec := range chunk { // WHERE IN
			args = append(args, rec.SongID, rec.DifficultyID)
		}

		if _, err := mysql.ExecContext(ctx, buf.String(), args...); err != nil {
			return fmt.Errorf("failed to bulk update charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type worldsendChartUpdateRecord struct {
	SongID    int
	LevelStar sql.NullInt64
	Attribute sql.NullString
	Notes     sql.NullInt64
}

// bulkUpdateMySQLWorldsendCharts 用のテンプレート（パフォーマンスのため事前にパースしておく）
var bulkUpdateWorldsendChartsTpl = template.Must(template.New("bulkUpdateWorldsendCharts").Parse(`
UPDATE worldsend_charts
SET
	level_star = CASE
		{{- range .Records }}
		WHEN song_id = ? THEN ?
		{{- end }}
		ELSE level_star
	END,
	attribute = CASE
		{{- range .Records }}
		WHEN song_id = ? THEN ?
		{{- end }}
		ELSE attribute
	END,
	notes = CASE
		{{- range .Records }}
		WHEN song_id = ? THEN ?
		{{- end }}
		ELSE notes
	END
WHERE song_id IN ({{ .Placeholders }})
`))

func bulkUpdateMySQLWorldsendCharts(ctx context.Context, mysql domainrepo.DBExecutor, records []worldsendChartUpdateRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		placeholders := strings.Repeat("?,", len(chunk))
		placeholders = placeholders[:len(placeholders)-1]

		var buf bytes.Buffer
		tplData := map[string]any{
			"Records":      chunk,
			"Placeholders": placeholders,
		}
		if err := bulkUpdateWorldsendChartsTpl.Execute(&buf, tplData); err != nil {
			return fmt.Errorf("failed to render bulk update worldsend_charts template (%d-%d): %w", start, end, err)
		}

		var args []any
		for _, rec := range chunk { // level_star
			args = append(args, rec.SongID, nullableInt(rec.LevelStar))
		}
		for _, rec := range chunk { // attribute
			args = append(args, rec.SongID, nullableString(rec.Attribute))
		}
		for _, rec := range chunk { // notes
			args = append(args, rec.SongID, nullableInt(rec.Notes))
		}
		for _, rec := range chunk { // WHERE IN
			args = append(args, rec.SongID)
		}

		if _, err := mysql.ExecContext(ctx, buf.String(), args...); err != nil {
			return fmt.Errorf("failed to bulk update worldsend_charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type syncAction int

const (
	actionSkip syncAction = iota
	actionInsert
	actionUpdate
)

const (
	majorUpdateLevelThreshold = 10.0
	majorUpdateLevelRange     = 0.5
)

func resolveChartUpdate(existing mysqlChart, exists bool, chart workspaceChart, opts SyncOptions) (float64, bool, sql.NullInt64, syncAction) {
	if !exists {
		// 挿入ロジック
		tConst := chart.Const
		tUnknown := chart.IsConstUnknown
		if opts.MajorUpdate {
			if chart.Const < majorUpdateLevelThreshold {
				tUnknown = false
			} else {
				tUnknown = true
			}
		}
		return tConst, tUnknown, chart.Notes, actionInsert
	}

	// 更新ロジック
	finalConst := existing.Const
	finalUnknown := existing.IsConstUnknown
	finalNotes := existing.Notes
	shouldUpdate := false

	if opts.MajorUpdate {
		if chart.Const < majorUpdateLevelThreshold {
			finalConst = chart.Const
			finalUnknown = false
			if existing.Const != finalConst || existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		} else {
			finalUnknown = true
			// 矛盾チェック
			rangeStart := chart.Const
			rangeEnd := chart.Const + majorUpdateLevelRange
			if existing.Const < rangeStart || existing.Const >= rangeEnd {
				finalConst = chart.Const
				shouldUpdate = true
			}

			if existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		}

		// MajorUpdate用のノーツチェック
		if !shouldUpdate {
			if !existing.Notes.Valid && chart.Notes.Valid {
				shouldUpdate = true
			}
		}
	} else {
		// 通常ロジック

		// 1. 詳細化 (不明 -> 既知) - 常に許可、定数を更新
		if existing.IsConstUnknown && !chart.IsConstUnknown {
			finalConst = chart.Const
			finalUnknown = false
			shouldUpdate = true
		} else {
			// 既知 -> 不明 への変更はブロック
			// それ以外（既知 -> 既知、不明 -> 不明）で、定数が一致し、かつノーツが新しい場合のみ更新
			isBlocked := !existing.IsConstUnknown && chart.IsConstUnknown

			if !isBlocked && existing.Const == chart.Const {
				if !existing.Notes.Valid && chart.Notes.Valid {
					shouldUpdate = true
				}
			}
		}
	}

	// ノーツ数の更新制御: 既存値が存在する場合、NULL値での上書きを防止
	if chart.Notes.Valid {
		finalNotes = chart.Notes
	}
	// 既存値がない場合のみ、新しい値（NULLでも）を受け入れる
	if !existing.Notes.Valid && chart.Notes.Valid {
		finalNotes = chart.Notes
	}

	if shouldUpdate {
		return finalConst, finalUnknown, finalNotes, actionUpdate
	}
	return finalConst, finalUnknown, finalNotes, actionSkip
}

func chartKey(songID, difficultyID int) string {
	return fmt.Sprintf("%d:%d", songID, difficultyID)
}

func nullableInt(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func nullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

type songInsertRecord struct {
	DisplayID   string
	Title       string
	Artist      string
	GenreID     sql.NullInt64
	BPM         sql.NullInt64
	ReleasedAt  sql.NullString
	OfficialIdx string
	Jacket      sql.NullString
	IsWorldsend int
}

func bulkUpsertMySQLSongs(ctx context.Context, mysql domainrepo.DBExecutor, records []songInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	const queryPrefix = `
INSERT INTO songs (
	display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
) VALUES `
	const querySuffix = `
ON DUPLICATE KEY UPDATE
	display_id = CASE
		WHEN display_id IS NULL OR display_id = '' THEN VALUES(display_id)
		ELSE display_id
	END,
	title = COALESCE(VALUES(title), title),
	artist = COALESCE(VALUES(artist), artist),
	genre_id = COALESCE(VALUES(genre_id), genre_id),
	bpm = COALESCE(VALUES(bpm), bpm),
	released_at = COALESCE(released_at, VALUES(released_at)),
	jacket = COALESCE(VALUES(jacket), jacket),
	is_worldsend = VALUES(is_worldsend),
	id = LAST_INSERT_ID(id)`

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*10)
		for i, rec := range chunk {
			values[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
			args = append(args,
				rec.DisplayID,
				rec.Title,
				rec.Artist,
				nullableInt(rec.GenreID),
				nullableInt(rec.BPM),
				nullableString(rec.ReleasedAt),
				rec.OfficialIdx,
				nullableString(rec.Jacket),
				rec.IsWorldsend,
				0,
			)
		}

		query := queryPrefix + strings.Join(values, ",") + querySuffix
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk upsert songs (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type chartInsertRecord struct {
	SongID        int
	DifficultyID  int
	Const         float64
	TargetUnknown bool
	Notes         sql.NullInt64
}

func bulkInsertMySQLCharts(ctx context.Context, mysql domainrepo.DBExecutor, records []chartInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	const queryPrefix = "INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes) VALUES "
	const querySuffix = ` ON DUPLICATE KEY UPDATE
        const = CASE
                WHEN is_const_unknown = 0 THEN const
                ELSE VALUES(const)
        END,
        is_const_unknown = CASE
                WHEN is_const_unknown = 0 AND VALUES(is_const_unknown) = 1 THEN 0
                ELSE VALUES(is_const_unknown)
        END,
        notes = COALESCE(VALUES(notes), notes)`

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*5)
		for i, rec := range chunk {
			values[i] = "(?, ?, ?, ?, ?)"
			args = append(args,
				rec.SongID,
				rec.DifficultyID,
				rec.Const,
				util.BoolToInt(rec.TargetUnknown),
				nullableInt(rec.Notes),
			)
		}

		query := queryPrefix + strings.Join(values, ",") + querySuffix
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk insert charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type worldsendChartInsertRecord struct {
	SongID    int
	LevelStar sql.NullInt64
	Attribute sql.NullString
	Notes     sql.NullInt64
}

func bulkInsertMySQLWorldsendCharts(ctx context.Context, mysql domainrepo.DBExecutor, records []worldsendChartInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	const queryPrefix = "INSERT INTO worldsend_charts (song_id, level_star, attribute, notes) VALUES "
	const querySuffix = ` ON DUPLICATE KEY UPDATE
        level_star = VALUES(level_star),
        attribute = VALUES(attribute),
        notes = COALESCE(VALUES(notes), notes)`

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*4)
		for i, rec := range chunk {
			values[i] = "(?, ?, ?, ?)"
			args = append(args,
				rec.SongID,
				nullableInt(rec.LevelStar),
				nullableString(rec.Attribute),
				nullableInt(rec.Notes),
			)
		}

		query := queryPrefix + strings.Join(values, ",") + querySuffix
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk insert worldsend_charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

func parseSQLStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}
