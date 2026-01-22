package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domainrepo "chunisupport-song-batch/internal/domain/repository"
	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/util"
	"chunisupport-song-batch/internal/workspace/songchart"

	"github.com/jmoiron/sqlx"
)

// additionalSongRecordForUpsert は追加楽曲のUPSERT用レコードです
type additionalSongRecordForUpsert struct {
	DisplayID   string  `db:"display_id"`
	Title       string  `db:"title"`
	Artist      string  `db:"artist"`
	GenreID     int     `db:"genre_id"`
	BPM         *int    `db:"bpm"`
	ReleasedAt  *string `db:"released_at"` // YYYY-MM-DD形式
	OfficialIdx string  `db:"official_idx"`
	Jacket      any     `db:"jacket"`
	IsWorldsEnd int     `db:"is_worldsend"`
}

// additionalChartRecordForUpsert は追加譜面のUPSERT用レコードです
type additionalChartRecordForUpsert struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
	Notes          *int    `db:"notes"`
}

// AdditionalSongsConsolidator は追加楽曲データソースの統合を処理します。
// 公式データに存在しない楽曲を補完する役割を持ち、
// 公式データに存在する楽曲（official_idx が一致するもの）は、追加データより公式データを優先します。
type AdditionalSongsConsolidator struct {
	db            domainrepo.ExtendedDBExecutor
	workspace     *songchart.SongChartWorkspace
	pwPepper      string
	data          *importer.AdditionalSongsData
	difficultyMap map[string]int
	genreIDByKey  map[string]int
}

// NewAdditionalSongsConsolidator は新しいAdditionalSongsConsolidatorのインスタンスを生成します。
func NewAdditionalSongsConsolidator(db domainrepo.ExtendedDBExecutor, difficultyRepo domainrepo.DifficultyRepository, genreRepo domainrepo.GenreRepository, workspace *songchart.SongChartWorkspace, pwPepper string, data *importer.AdditionalSongsData) *AdditionalSongsConsolidator {
	difficultyMap, err := loadDifficultyMap(context.Background(), difficultyRepo)
	if err != nil {
		slog.Warn("Failed to load chart difficulties; falling back to defaults", "error", err)
		difficultyMap = defaultDifficultyMap()
	}
	genreMap, err := loadGenreMap(context.Background(), genreRepo)
	if err != nil {
		slog.Warn("Failed to load genres; additional_songs genre assignment may be incomplete", "error", err)
		genreMap = make(map[string]int)
	}

	return &AdditionalSongsConsolidator{
		db:            db,
		workspace:     workspace,
		pwPepper:      pwPepper,
		data:          data,
		difficultyMap: difficultyMap,
		genreIDByKey:  genreMap,
	}
}

// Consolidate は追加楽曲データをワークスペースへ反映します。
// 公式データに存在する楽曲（official_idx が一致するもの）は、追加データより公式データを優先します。
func (c *AdditionalSongsConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || (len(c.data.Songs) == 0 && len(c.data.Charts) == 0 && len(c.data.WECharts) == 0) {
		slog.Warn("Additional songs data is empty; skipping consolidation")
		return nil
	}

	// ワークスペースに既に登録されている official_idx を取得
	existingIdxs, err := c.loadExistingOfficialIdxs(ctx)
	if err != nil {
		return err
	}

	// Step 1: 追加楽曲(songs)の処理 - 公式データに存在しない楽曲のみを追加
	if len(c.data.Songs) > 0 {
		if err := c.consolidateSongs(ctx, existingIdxs); err != nil {
			return err
		}
	}

	// Step 2: 追加譜面(charts)の処理 - 既存楽曲へのULTIMA追加など
	if len(c.data.Charts) > 0 {
		if err := c.consolidateCharts(ctx, existingIdxs); err != nil {
			return err
		}
	}

	// Step 3: WORLD'S END譜面(we_charts)の処理
	if len(c.data.WECharts) > 0 {
		if err := c.consolidateWECharts(ctx, existingIdxs); err != nil {
			return err
		}
	}

	slog.Info("Additional songs data consolidation completed",
		"songs", len(c.data.Songs), "charts", len(c.data.Charts), "we_charts", len(c.data.WECharts))
	return nil
}

// consolidateSongs は追加楽曲をワークスペースに反映します
func (c *AdditionalSongsConsolidator) consolidateSongs(ctx context.Context, existingIdxs map[string]struct{}) error {
	// Step 1: 公式データに存在しない楽曲のみを抽出してUPSERT
	songsToUpsert, seenOfficialIdx := c.prepareSongsForUpsert(existingIdxs)
	if len(songsToUpsert) == 0 {
		slog.Info("No new songs to add from additional_songs data (all songs already exist in official data)")
		return nil
	}

	if err := c.bulkUpsertSongs(ctx, songsToUpsert); err != nil {
		return err
	}
	slog.Info("Bulk upserted additional songs", "count", len(songsToUpsert))

	// Step 2: UPSERTした楽曲のIDを取得
	songIDs, err := c.fetchSongIDsByOfficialIdx(ctx, seenOfficialIdx)
	if err != nil {
		return err
	}

	// Step 3: 通常チャート情報を一括でUPSERT（追加楽曲のチャート）
	chartsToUpsert := c.prepareChartsFromSongsForUpsert(songIDs)
	if len(chartsToUpsert) > 0 {
		if err := c.bulkUpsertCharts(ctx, chartsToUpsert); err != nil {
			return err
		}
		slog.Info("Bulk upserted additional charts from songs", "count", len(chartsToUpsert))
	}

	return nil
}

// consolidateCharts は追加譜面（既存楽曲へのULTIMA追加など）をワークスペースに反映します
func (c *AdditionalSongsConsolidator) consolidateCharts(ctx context.Context, existingIdxs map[string]struct{}) error {
	// 追加譜面の対象となる楽曲IDを取得（公式データに存在するもののみ）
	chartOfficialIdxs := make(map[string]struct{})
	for _, chart := range c.data.Charts {
		officialID := strings.TrimSpace(chart.ID)
		if officialID == "" {
			continue
		}
		// 公式データに存在するもののみを対象とする
		if _, exists := existingIdxs[officialID]; exists {
			chartOfficialIdxs[officialID] = struct{}{}
		}
	}

	if len(chartOfficialIdxs) == 0 {
		slog.Info("No charts to add (target songs not found in workspace)")
		return nil
	}

	// 楽曲IDを取得
	songIDs, err := c.fetchSongIDsByOfficialIdx(ctx, chartOfficialIdxs)
	if err != nil {
		return err
	}

	// 既存のチャート情報を取得（公式データを尊重するため）
	existingCharts, err := c.loadExistingCharts(ctx, songIDs)
	if err != nil {
		return err
	}

	// 追加譜面を準備
	chartsToUpsert := c.prepareAdditionalChartsForUpsert(songIDs, existingCharts)
	if len(chartsToUpsert) == 0 {
		slog.Info("No additional charts to add (all charts already exist)")
		return nil
	}

	if err := c.bulkUpsertCharts(ctx, chartsToUpsert); err != nil {
		return err
	}
	slog.Info("Bulk upserted additional charts", "count", len(chartsToUpsert))

	return nil
}

// loadExistingOfficialIdxs はワークスペースに既に登録されている official_idx のセットを取得します
func (c *AdditionalSongsConsolidator) loadExistingOfficialIdxs(ctx context.Context) (map[string]struct{}, error) {
	type songRecord struct {
		OfficialIdx string `db:"official_idx"`
	}

	var rows []songRecord
	query := `SELECT official_idx FROM songs WHERE official_idx IS NOT NULL`

	if err := c.workspace.DB().SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to load existing official_idx from workspace: %w", err)
	}

	result := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.OfficialIdx)
		if key == "" {
			continue
		}
		result[key] = struct{}{}
	}

	slog.Debug("Loaded existing official_idx from workspace", "count", len(result))
	return result, nil
}

// loadExistingCharts は指定された楽曲IDのチャート情報を取得します
func (c *AdditionalSongsConsolidator) loadExistingCharts(ctx context.Context, songIDs map[string]int) (map[string]struct{}, error) {
	if len(songIDs) == 0 {
		return make(map[string]struct{}), nil
	}

	songIDList := make([]int, 0, len(songIDs))
	for _, id := range songIDs {
		songIDList = append(songIDList, id)
	}

	query, args, err := sqlx.In(`SELECT song_id, difficulty_id FROM charts WHERE song_id IN (?)`, songIDList)
	if err != nil {
		return nil, fmt.Errorf("failed to build query for existing charts: %w", err)
	}

	type chartRecord struct {
		SongID       int `db:"song_id"`
		DifficultyID int `db:"difficulty_id"`
	}

	var records []chartRecord
	if err := c.workspace.DB().SelectContext(ctx, &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to load existing charts: %w", err)
	}

	// song_id|difficulty_id をキーとするマップを作成
	result := make(map[string]struct{}, len(records))
	for _, r := range records {
		key := fmt.Sprintf("%d|%d", r.SongID, r.DifficultyID)
		result[key] = struct{}{}
	}

	return result, nil
}

func (c *AdditionalSongsConsolidator) prepareSongsForUpsert(existingIdxs map[string]struct{}) ([]additionalSongRecordForUpsert, map[string]struct{}) {
	var songsToUpsert []additionalSongRecordForUpsert
	seenOfficialIdx := make(map[string]struct{})
	var skippedCount int

	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.ID)
		if officialID == "" {
			slog.Warn("Skipping additional song without official_idx", "title", song.Title, "artist", song.Artist)
			continue
		}

		// 公式データに既に存在する場合はスキップ
		if _, exists := existingIdxs[officialID]; exists {
			skippedCount++
			continue
		}

		genreID := c.lookupGenreID(song.Genre)
		if genreID == 0 {
			slog.Warn("Skipping additional song with unknown genre", "title", song.Title, "genre", song.Genre)
			continue
		}

		image := normalizeImage(strings.TrimSpace(song.Img))
		displayID := GenerateDisplayID()

		// リリース日をパース（YYYY-MM-DD形式に正規化）
		var releaseDate *string
		if song.Release != "" {
			if normalizedDate, err := normalizeReleaseDate(song.Release); err == nil {
				releaseDate = &normalizedDate
			} else {
				slog.Warn("Failed to parse release date", "release", song.Release, "title", song.Title, "error", err)
			}
		}

		songsToUpsert = append(songsToUpsert, additionalSongRecordForUpsert{
			DisplayID:   displayID,
			Title:       strings.TrimSpace(song.Title),
			Artist:      strings.TrimSpace(song.Artist),
			GenreID:     genreID,
			BPM:         song.BPM,
			ReleasedAt:  releaseDate,
			OfficialIdx: officialID,
			Jacket:      nullIfEmpty(image),
			IsWorldsEnd: 0, // 追加楽曲はWorld's Endではない
		})
		seenOfficialIdx[officialID] = struct{}{}
	}

	if skippedCount > 0 {
		slog.Info("Skipped songs already in official data", "count", skippedCount)
	}

	return songsToUpsert, seenOfficialIdx
}

func (c *AdditionalSongsConsolidator) bulkUpsertSongs(ctx context.Context, songs []additionalSongRecordForUpsert) error {
	query := `
INSERT INTO songs (
	display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
) VALUES (
	:display_id, :title, :artist, :genre_id, :bpm, :released_at, :official_idx, :jacket, :is_worldsend, 0
)
ON CONFLICT(official_idx) DO UPDATE SET
	display_id = excluded.display_id,
	title = excluded.title,
	artist = excluded.artist,
	genre_id = excluded.genre_id,
	bpm = COALESCE(excluded.bpm, songs.bpm),
	released_at = COALESCE(excluded.released_at, songs.released_at),
	jacket = COALESCE(excluded.jacket, songs.jacket),
	is_worldsend = excluded.is_worldsend
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, songs)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert additional songs: %w", err)
	}
	return nil
}

func (c *AdditionalSongsConsolidator) fetchSongIDsByOfficialIdx(ctx context.Context, officialIdxs map[string]struct{}) (map[string]int, error) {
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

	type songIDRecord struct {
		ID          int    `db:"id"`
		OfficialIdx string `db:"official_idx"`
	}

	var records []songIDRecord
	if err := c.workspace.DB().SelectContext(ctx, &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch song IDs by official_idx: %w", err)
	}

	result := make(map[string]int, len(records))
	for _, r := range records {
		result[r.OfficialIdx] = r.ID
	}
	return result, nil
}

// prepareChartsFromSongsForUpsert は追加楽曲(songs)のチャート情報を準備します
func (c *AdditionalSongsConsolidator) prepareChartsFromSongsForUpsert(songIDs map[string]int) []additionalChartRecordForUpsert {
	var chartsToUpsert []additionalChartRecordForUpsert

	// 難易度とデータ取得関数のマッピング
	type difficultyGetter struct {
		name     string
		getConst func(s *importer.AdditionalSong) float64
		getUK    func(s *importer.AdditionalSong) bool
		getNotes func(s *importer.AdditionalSong) *int
		hasChart func(s *importer.AdditionalSong) bool // 譜面が存在するかの判定
	}

	difficultyGetters := []difficultyGetter{
		{
			name:     "BASIC",
			getConst: func(s *importer.AdditionalSong) float64 { return s.Bas },
			getUK:    func(s *importer.AdditionalSong) bool { return s.BasUK },
			getNotes: func(s *importer.AdditionalSong) *int { return s.BasNt },
			hasChart: func(s *importer.AdditionalSong) bool { return s.Bas > 0 },
		},
		{
			name:     "ADVANCED",
			getConst: func(s *importer.AdditionalSong) float64 { return s.Adv },
			getUK:    func(s *importer.AdditionalSong) bool { return s.AdvUK },
			getNotes: func(s *importer.AdditionalSong) *int { return s.AdvNt },
			hasChart: func(s *importer.AdditionalSong) bool { return s.Adv > 0 },
		},
		{
			name:     "EXPERT",
			getConst: func(s *importer.AdditionalSong) float64 { return s.Exp },
			getUK:    func(s *importer.AdditionalSong) bool { return s.ExpUK },
			getNotes: func(s *importer.AdditionalSong) *int { return s.ExpNt },
			hasChart: func(s *importer.AdditionalSong) bool { return s.Exp > 0 },
		},
		{
			name:     "MASTER",
			getConst: func(s *importer.AdditionalSong) float64 { return s.Mas },
			getUK:    func(s *importer.AdditionalSong) bool { return s.MasUK },
			getNotes: func(s *importer.AdditionalSong) *int { return s.MasNt },
			hasChart: func(s *importer.AdditionalSong) bool { return s.Mas > 0 },
		},
		{
			name:     "ULTIMA",
			getConst: func(s *importer.AdditionalSong) float64 { return s.Ult },
			getUK:    func(s *importer.AdditionalSong) bool { return s.UltUK },
			getNotes: func(s *importer.AdditionalSong) *int { return s.UltNt },
			hasChart: func(s *importer.AdditionalSong) bool { return s.Ult > 0 },
		},
	}

	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		for _, getter := range difficultyGetters {
			if !getter.hasChart(&song) {
				continue
			}

			diffID := c.difficultyMap[getter.name]
			if diffID == 0 {
				slog.Warn("Unknown difficulty in workspace; skipping additional chart",
					"difficulty", getter.name, "song", song.Title)
				continue
			}

			constValue := getter.getConst(&song)
			isConstUnknown := getter.getUK(&song)
			notes := getter.getNotes(&song)

			chartsToUpsert = append(chartsToUpsert, additionalChartRecordForUpsert{
				SongID:         songID,
				DifficultyID:   diffID,
				Const:          constValue,
				IsConstUnknown: util.BoolToInt(isConstUnknown),
				Notes:          notes,
			})
		}
	}

	return chartsToUpsert
}

// prepareAdditionalChartsForUpsert は追加譜面(charts)をUPSERT用に準備します
// 既存のチャートがある場合はスキップ（公式データを尊重）
func (c *AdditionalSongsConsolidator) prepareAdditionalChartsForUpsert(songIDs map[string]int, existingCharts map[string]struct{}) []additionalChartRecordForUpsert {
	var chartsToUpsert []additionalChartRecordForUpsert

	for _, chart := range c.data.Charts {
		officialID := strings.TrimSpace(chart.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		// 難易度をマップ
		diffName := strings.ToUpper(strings.TrimSpace(chart.Diff))
		diffID := c.difficultyMap[diffName]
		if diffID == 0 {
			slog.Warn("Unknown difficulty in additional_charts; skipping",
				"difficulty", chart.Diff, "songID", officialID)
			continue
		}

		// 既存チャートがある場合はスキップ（公式データを尊重）
		chartKey := fmt.Sprintf("%d|%d", songID, diffID)
		if _, exists := existingCharts[chartKey]; exists {
			slog.Debug("Skipping additional chart (already exists in official data)",
				"songID", officialID, "difficulty", diffName)
			continue
		}

		chartsToUpsert = append(chartsToUpsert, additionalChartRecordForUpsert{
			SongID:         songID,
			DifficultyID:   diffID,
			Const:          chart.Const,
			IsConstUnknown: util.BoolToInt(chart.CsUK),
			Notes:          chart.Notes,
		})
	}

	return chartsToUpsert
}

func (c *AdditionalSongsConsolidator) bulkUpsertCharts(ctx context.Context, charts []additionalChartRecordForUpsert) error {
	query := `
INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
VALUES (:song_id, :difficulty_id, :const, :is_const_unknown, :notes)
ON CONFLICT(song_id, difficulty_id) DO UPDATE SET
	const = excluded.const,
	is_const_unknown = excluded.is_const_unknown,
	notes = COALESCE(excluded.notes, charts.notes)
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, charts)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert additional charts: %w", err)
	}
	return nil
}

func (c *AdditionalSongsConsolidator) lookupGenreID(genre string) int {
	if len(c.genreIDByKey) == 0 {
		return 0
	}
	key := strings.TrimSpace(strings.ToUpper(genre))
	return c.genreIDByKey[key]
}

// consolidateWECharts はWORLD'S END譜面をワークスペースに反映します
func (c *AdditionalSongsConsolidator) consolidateWECharts(ctx context.Context, existingIdxs map[string]struct{}) error {
	// WORLD'S END楽曲を準備してUPSERT
	weRecords, seenOfficialIdx := c.prepareWESongsForUpsert(existingIdxs)
	if len(weRecords) == 0 {
		slog.Info("No new WORLD'S END songs to add")
		return nil
	}

	if err := c.bulkUpsertSongs(ctx, weRecords); err != nil {
		return err
	}
	slog.Info("Bulk upserted WORLD'S END songs", "count", len(weRecords))

	// UPSERTした楽曲のIDを取得
	songIDs, err := c.fetchSongIDsByOfficialIdx(ctx, seenOfficialIdx)
	if err != nil {
		return err
	}

	// WORLD'S END譜面情報をUPSERT
	weChartsToUpsert := c.prepareWEChartsForUpsert(songIDs)
	if len(weChartsToUpsert) > 0 {
		if err := c.bulkUpsertWECharts(ctx, weChartsToUpsert); err != nil {
			return err
		}
		slog.Info("Bulk upserted WORLD'S END charts", "count", len(weChartsToUpsert))
	}

	return nil
}

// prepareWESongsForUpsert はWORLD'S END楽曲のUPSERT用レコードを準備します
func (c *AdditionalSongsConsolidator) prepareWESongsForUpsert(existingIdxs map[string]struct{}) ([]additionalSongRecordForUpsert, map[string]struct{}) {
	var songsToUpsert []additionalSongRecordForUpsert
	seenOfficialIdx := make(map[string]struct{})
	var skippedCount int

	for _, weChart := range c.data.WECharts {
		officialID := strings.TrimSpace(weChart.ID)
		if officialID == "" {
			slog.Warn("Skipping WORLD'S END song without ID", "title", weChart.Title)
			continue
		}

		// 既に存在する場合はスキップ
		if _, exists := existingIdxs[officialID]; exists {
			skippedCount++
			continue
		}

		genreID := c.lookupGenreID(weChart.Genre)
		if genreID == 0 {
			slog.Warn("Skipping WORLD'S END song with unknown genre", "title", weChart.Title, "genre", weChart.Genre)
			continue
		}

		image := normalizeImage(strings.TrimSpace(weChart.Img))
		displayID := GenerateDisplayID()

		// リリース日をパース（YYYY-MM-DD形式に正規化）
		var releaseDate *string
		if weChart.Release != "" {
			if normalizedDate, err := normalizeReleaseDate(weChart.Release); err == nil {
				releaseDate = &normalizedDate
			} else {
				slog.Warn("Failed to parse release date for WORLD'S END song", "release", weChart.Release, "title", weChart.Title, "error", err)
			}
		}

		songsToUpsert = append(songsToUpsert, additionalSongRecordForUpsert{
			DisplayID:   displayID,
			Title:       strings.TrimSpace(weChart.Title),
			Artist:      strings.TrimSpace(weChart.Artist),
			GenreID:     genreID,
			BPM:         nil, // WORLD'S ENDはBPM情報なし
			ReleasedAt:  releaseDate,
			OfficialIdx: officialID,
			Jacket:      nullIfEmpty(image),
			IsWorldsEnd: 1, // WORLD'S ENDフラグ
		})
		seenOfficialIdx[officialID] = struct{}{}
	}

	if skippedCount > 0 {
		slog.Info("Skipped WORLD'S END songs already in database", "count", skippedCount)
	}

	return songsToUpsert, seenOfficialIdx
}

// weChartRecordForUpsert はWORLD'S END譜面のUPSERT用レコードです
type weChartRecordForUpsert struct {
	SongID  int     `db:"song_id"`
	WEStar  *int    `db:"we_star"`
	WEKanji *string `db:"we_kanji"`
	Notes   *int    `db:"notes"`
}

// prepareWEChartsForUpsert はWORLD'S END譜面のUPSERT用レコードを準備します
func (c *AdditionalSongsConsolidator) prepareWEChartsForUpsert(songIDs map[string]int) []weChartRecordForUpsert {
	var chartsToUpsert []weChartRecordForUpsert

	for _, weChart := range c.data.WECharts {
		officialID := strings.TrimSpace(weChart.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		// we_kanjiをポインタ型に変換
		var weKanji *string
		kanjiStr := strings.TrimSpace(weChart.WEKanji)
		if kanjiStr != "" {
			weKanji = &kanjiStr
		}

		chartsToUpsert = append(chartsToUpsert, weChartRecordForUpsert{
			SongID:  songID,
			WEStar:  weChart.WEStar,
			WEKanji: weKanji,
			Notes:   weChart.Notes,
		})
	}

	return chartsToUpsert
}

// bulkUpsertWECharts はWORLD'S END譜面を一括でUPSERTします
func (c *AdditionalSongsConsolidator) bulkUpsertWECharts(ctx context.Context, charts []weChartRecordForUpsert) error {
	query := `
INSERT INTO worldsend_charts (song_id, we_star, we_kanji, notes)
VALUES (:song_id, :we_star, :we_kanji, :notes)
ON CONFLICT(song_id) DO UPDATE SET
	we_star = COALESCE(excluded.we_star, worldsend_charts.we_star),
	we_kanji = COALESCE(excluded.we_kanji, worldsend_charts.we_kanji),
	notes = COALESCE(excluded.notes, worldsend_charts.notes)
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, charts)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert WORLD'S END charts: %w", err)
	}
	return nil
}

// normalizeReleaseDate はリリース日文字列をYYYY-MM-DD形式に正規化します
// フォーマット: YYYY/MM/DD または YYYY-MM-DD
func normalizeReleaseDate(dateStr string) (string, error) {
	// 複数のフォーマットを試行
	formats := []string{
		"2006/01/02",
		"2006-01-02",
		"2006/1/2",
		"2006-1-2",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// YYYY-MM-DD形式で返却
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unable to parse date: %s", dateStr)
}
