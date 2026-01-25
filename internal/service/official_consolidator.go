package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"chunisupport-song-batch/internal/domain/difficulty"
	"chunisupport-song-batch/internal/domain/entity"
	domainrepo "chunisupport-song-batch/internal/domain/repository"
	vo "chunisupport-song-batch/internal/domain/valueobject"
	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/infra/models"
	"chunisupport-song-batch/internal/util"
	"chunisupport-song-batch/internal/workspace/songchart"

	"github.com/jmoiron/sqlx"
)

type difficultyInfo struct {
	ID   int
	Name string
}

// diffIDToDomainDifficultyID はDBのdifficultyIDをドメインのdifficulty.IDに変換します
func diffIDToDomainDifficultyID(id int) difficulty.ID {
	return difficulty.ID(id)
}

// OfficialConsolidator は公式データソースの統合を処理します。
type OfficialConsolidator struct {
	db            domainrepo.ExtendedDBExecutor
	workspace     *songchart.SongChartWorkspace
	pwPepper      string
	data          *importer.OfficialData
	difficultyMap map[string]int
	genreIDByKey  map[string]int
}

// NewOfficialConsolidator は新しいOfficialConsolidatorのインスタンスを生成します。
func NewOfficialConsolidator(db domainrepo.ExtendedDBExecutor, difficultyRepo domainrepo.DifficultyRepository, genreRepo domainrepo.GenreRepository, workspace *songchart.SongChartWorkspace, pwPepper string, data *importer.OfficialData) *OfficialConsolidator {
	difficultyMap, err := loadDifficultyMap(context.Background(), difficultyRepo)
	if err != nil {
		slog.Warn("Failed to load chart difficulties; falling back to defaults", "error", err)
		difficultyMap = defaultDifficultyMap()
	}
	genreMap, err := loadGenreMap(context.Background(), genreRepo)
	if err != nil {
		slog.Warn("Failed to load genres; official genre assignment may be incomplete", "error", err)
		genreMap = make(map[string]int)
	}

	return &OfficialConsolidator{
		db:            db,
		workspace:     workspace,
		pwPepper:      pwPepper,
		data:          data,
		difficultyMap: difficultyMap,
		genreIDByKey:  genreMap,
	}
}

// Consolidate は公式データをワークスペースへ反映します。
func (c *OfficialConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(*c.data) == 0 {
		slog.Warn("Official data is empty; skipping consolidation")
		return nil
	}

	existingActiveSongs, err := c.loadActiveOfficialSongs(ctx)
	if err != nil {
		return err
	}

	// official_idx の大規模変更を検知
	if err := c.detectMassiveIdxChange(ctx, existingActiveSongs); err != nil {
		return err
	}

	// Step 1: 楽曲情報を一括でUPSERT
	songsToUpsert, seenOfficialIdx := c.prepareSongsForUpsert()
	if len(songsToUpsert) == 0 {
		slog.Warn("No valid songs to process from official data")
		return nil
	}

	if err := c.bulkUpsertSongs(ctx, songsToUpsert); err != nil {
		return err
	}
	slog.Info("Bulk upserted songs", "count", len(songsToUpsert))

	// Step 2: UPSERTした楽曲のIDを取得
	songIDs, err := c.fetchSongIDsByOfficialIdx(ctx, seenOfficialIdx)
	if err != nil {
		return err
	}

	// Step 3: 通常チャート情報を一括でUPSERT
	chartsToUpsert := c.prepareChartsForUpsert(songIDs)
	if len(chartsToUpsert) > 0 {
		if err := c.bulkUpsertCharts(ctx, chartsToUpsert); err != nil {
			return err
		}
		slog.Info("Bulk upserted charts", "count", len(chartsToUpsert))
	}

	// Step 4: World's End チャート情報を一括でUPSERT
	worldsendChartsToUpsert := c.prepareWorldsendChartsForUpsert(songIDs)
	if len(worldsendChartsToUpsert) > 0 {
		if err := c.bulkUpsertWorldsendCharts(ctx, worldsendChartsToUpsert); err != nil {
			return err
		}
		slog.Info("Bulk upserted worldsend_charts", "count", len(worldsendChartsToUpsert))
	}

	slog.Info("Official data consolidation completed", "songs", len(songsToUpsert))
	return nil
}

func (c *OfficialConsolidator) prepareSongsForUpsert() ([]*models.SongModelForUpsert, map[string]struct{}) {
	var songsToUpsert []*models.SongModelForUpsert
	seenOfficialIdx := make(map[string]struct{})

	for _, officialSong := range *c.data {
		officialID := strings.TrimSpace(officialSong.ID)
		if officialID == "" {
			slog.Warn("Skipping official song without official_idx", "title", officialSong.Title, "artist", officialSong.Artist)
			continue
		}

		genreID := c.lookupGenreID(officialSong.Catname)
		if genreID == 0 {
			slog.Warn("Skipping official song with unknown genre", "title", officialSong.Title, "catname", officialSong.Catname)
			continue
		}

		// ドメインエンティティを使用してSongを生成
		song, err := entity.NewSongFromOfficial(&officialSong, genreID)
		if err != nil {
			slog.Warn("Skipping invalid official song", "title", officialSong.Title, "error", err)
			continue
		}

		songsToUpsert = append(songsToUpsert, models.FromSongEntityForUpsert(song))
		seenOfficialIdx[officialID] = struct{}{}
	}
	return songsToUpsert, seenOfficialIdx
}

func (c *OfficialConsolidator) bulkUpsertSongs(ctx context.Context, songs []*models.SongModelForUpsert) error {
	query := `
INSERT INTO songs (
	display_id, title, artist, genre_id, official_idx, jacket, is_worldsend, is_deleted
) VALUES (
	:display_id, :title, :artist, :genre_id, :official_idx, :jacket, :is_worldsend, 0
)
ON CONFLICT(official_idx) DO UPDATE SET
	display_id = excluded.display_id,
	title = excluded.title,
	artist = excluded.artist,
	genre_id = excluded.genre_id,
	jacket = excluded.jacket,
	is_worldsend = excluded.is_worldsend
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, songs)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert songs: %w", err)
	}
	return nil
}

func (c *OfficialConsolidator) fetchSongIDsByOfficialIdx(ctx context.Context, officialIdxs map[string]struct{}) (map[string]int, error) {
	if len(officialIdxs) == 0 {
		return make(map[string]int), nil
	}

	idxList := slices.Collect(maps.Keys(officialIdxs))

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

// detectMassiveIdxChange は official_idx の大規模変更を検知します。
// 既存楽曲の多くが official_idx でマッチしない場合、エラーを返します。
func (c *OfficialConsolidator) detectMassiveIdxChange(ctx context.Context, existingActiveSongs map[string]int) error {
	// 既存の楽曲一覧を取得（削除されていないもののみ）
	var existingSongs []struct {
		ID          int    `db:"id"`
		Title       string `db:"title"`
		Artist      string `db:"artist"`
		OfficialIdx string `db:"official_idx"`
		IsWorldsEnd int    `db:"is_worldsend"`
	}

	// 既存楽曲が少ない場合はDBから再取得（より多くの情報を含む）
	if len(existingActiveSongs) < 10 {
		slog.Debug("Skipping idx change detection due to small dataset", "existing_count", len(existingActiveSongs))
		return nil
	}

	idxList := make([]string, 0, len(existingActiveSongs))
	for idx := range existingActiveSongs {
		idxList = append(idxList, idx)
	}

	query, args, err := sqlx.In(`SELECT id, title, artist, official_idx, is_worldsend FROM songs WHERE official_idx IN (?) AND is_deleted = 0`, idxList)
	if err != nil {
		return fmt.Errorf("failed to build query for idx change detection: %w", err)
	}

	// ExtendedDBExecutor インターフェースを通じて MySQL から楽曲詳細を取得
	if err := c.db.SelectContext(ctx, &existingSongs, c.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("failed to load existing songs for idx change detection: %w", err)
	}

	// 新しいデータの official_idx のセットを構築
	newOfficialIdxSet := make(map[string]struct{}, len(*c.data))
	// title+artist のマップも構築（マッチング用）
	newSongsByTitleArtist := make(map[string]string, len(*c.data))

	for _, song := range *c.data {
		officialID := strings.TrimSpace(song.ID)
		if officialID == "" {
			continue
		}
		title := strings.TrimSpace(song.Title)
		artist := strings.TrimSpace(song.Artist)

		isWorldsEnd := strings.TrimSpace(song.WeKanji) != "" || strings.TrimSpace(song.WeStar) != ""
		isWEInt := util.BoolToInt(isWorldsEnd)

		newOfficialIdxSet[officialID] = struct{}{}
		key := fmt.Sprintf("%s|||%s|||%d", title, artist, isWEInt)
		newSongsByTitleArtist[key] = officialID
	}

	// マッチング状況を分析
	var missingByIdx int           // official_idx でマッチしない数
	var matchByTitleArtist int     // title+artist でマッチする数
	var suspiciousChanges []string // 疑わしい変更のリスト

	for _, existing := range existingSongs {
		// official_idx でマッチするかチェック
		if _, found := newOfficialIdxSet[existing.OfficialIdx]; found {
			continue // 正常にマッチ
		}

		missingByIdx++

		// title+artist でマッチするかチェック
		key := fmt.Sprintf("%s|||%s|||%d", existing.Title, existing.Artist, existing.IsWorldsEnd)
		if newIdx, found := newSongsByTitleArtist[key]; found {
			matchByTitleArtist++
			suspiciousChanges = append(suspiciousChanges,
				fmt.Sprintf("  - '%s' by '%s': %s → %s",
					existing.Title, existing.Artist, existing.OfficialIdx, newIdx))
		}
	}

	// 閾値判定
	const absoluteThreshold = 10    // 絶対数での閾値
	const percentageThreshold = 0.2 // 20%の閾値

	changedPercentage := float64(matchByTitleArtist) / float64(len(existingSongs))

	if matchByTitleArtist >= absoluteThreshold || changedPercentage >= percentageThreshold {
		// 異常検知：大規模な official_idx 変更
		slog.Error("Massive official_idx change detected",
			"existing_songs", len(existingSongs),
			"missing_by_idx", missingByIdx,
			"match_by_title_artist", matchByTitleArtist,
			"changed_percentage", fmt.Sprintf("%.1f%%", changedPercentage*100))

		// 変更の詳細を出力（最大20件）
		if len(suspiciousChanges) > 0 {
			displayCount := min(len(suspiciousChanges), 20)
			slog.Error("Detected official_idx changes (showing first 20):")
			for i := 0; i < displayCount; i++ {
				slog.Error(suspiciousChanges[i])
			}
			if len(suspiciousChanges) > 20 {
				slog.Error(fmt.Sprintf("  ... and %d more", len(suspiciousChanges)-20))
			}
		}

		return fmt.Errorf(
			"massive official_idx change detected: %d songs (%.1f%%) would have their official_idx changed. "+
				"This requires manual intervention. Please review the changes and run a special migration process if needed",
			matchByTitleArtist, changedPercentage*100)
	}

	// 少数の変更は警告のみ
	if matchByTitleArtist > 0 {
		slog.Warn("Detected minor official_idx changes",
			"count", matchByTitleArtist,
			"percentage", fmt.Sprintf("%.1f%%", changedPercentage*100))
		for _, change := range suspiciousChanges {
			slog.Warn(change)
		}
	}

	return nil
}

func (c *OfficialConsolidator) prepareChartsForUpsert(songIDs map[string]int) []*models.ChartModelForUpsert {
	var chartsToUpsert []*models.ChartModelForUpsert

	levelByDifficultyKey := map[string]func(s *importer.OfficialSong) string{
		"BASIC":    func(s *importer.OfficialSong) string { return s.LevBas },
		"ADVANCED": func(s *importer.OfficialSong) string { return s.LevAdv },
		"EXPERT":   func(s *importer.OfficialSong) string { return s.LevExp },
		"MASTER":   func(s *importer.OfficialSong) string { return s.LevMas },
		"ULTIMA":   func(s *importer.OfficialSong) string { return s.LevUlt },
	}

	for _, song := range *c.data {
		officialID := strings.TrimSpace(song.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		// ドメインエンティティを使用してWORLD'S END判定
		if entity.DetermineIsWorldsEnd(&song) {
			continue
		}

		for diffName, getLevel := range levelByDifficultyKey {
			levelStr := strings.TrimSpace(getLevel(&song))
			if levelStr == "" {
				continue
			}

			diffID := c.difficultyMap[diffName]
			if diffID == 0 {
				slog.Warn("Unknown difficulty in workspace; skipping official chart", "difficulty", diffName, "song", song.Title)
				continue
			}

			// 値オブジェクトを使用してレベルをパース
			level, err := vo.ParseLevel(levelStr)
			if err != nil {
				slog.Warn("Failed to parse official level", "level", levelStr, "difficulty", diffName, "error", err)
				continue
			}

			// ドメインエンティティを作成してモデルに変換
			chart := entity.NewChart(songID, diffIDToDomainDifficultyID(diffID), level, level.IsConstUnknown())
			chartsToUpsert = append(chartsToUpsert, models.FromChartEntityForUpsert(chart))
		}
	}
	return chartsToUpsert
}

func (c *OfficialConsolidator) bulkUpsertCharts(ctx context.Context, charts []*models.ChartModelForUpsert) error {
	query := `
INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown)
VALUES (:song_id, :difficulty_id, :const, :is_const_unknown)
ON CONFLICT(song_id, difficulty_id) DO UPDATE SET
	const = excluded.const,
	is_const_unknown = excluded.is_const_unknown
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, charts)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert charts: %w", err)
	}
	return nil
}

func (c *OfficialConsolidator) prepareWorldsendChartsForUpsert(songIDs map[string]int) []*models.WorldsEndChartModelForUpsert {
	var chartsToUpsert []*models.WorldsEndChartModelForUpsert

	for _, song := range *c.data {
		officialID := strings.TrimSpace(song.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		// ドメインエンティティを使用してWORLD'S END判定
		if !entity.DetermineIsWorldsEnd(&song) {
			continue
		}

		// 値オブジェクトを使用してWE星数とWE漢字を処理
		weStar, err := vo.ParseWeStarFromOfficial(song.WeStar)
		if err != nil {
			slog.Warn("Failed to parse we_star", "we_star", song.WeStar, "song", song.Title, "error", err)
		}

		weKanji := vo.NewWeKanji(song.WeKanji)
		if !weKanji.IsEmpty() && len([]rune(song.WeKanji)) > 1 {
			slog.Warn("we_kanji longer than 1 character, truncating", "we_kanji", song.WeKanji, "song", song.Title)
		}

		// ドメインエンティティを作成してモデルに変換
		weChart := entity.NewWorldsEndChart(songID, weStar, weKanji)
		chartsToUpsert = append(chartsToUpsert, models.FromWorldsEndChartEntityForUpsert(weChart))
	}

	return chartsToUpsert
}

func (c *OfficialConsolidator) bulkUpsertWorldsendCharts(ctx context.Context, charts []*models.WorldsEndChartModelForUpsert) error {
	query := `
INSERT INTO worldsend_charts (song_id, we_star, we_kanji)
VALUES (:song_id, :we_star, :we_kanji)
ON CONFLICT(song_id) DO UPDATE SET
	we_star = excluded.we_star,
	we_kanji = excluded.we_kanji
`
	_, err := sqlx.NamedExecContext(ctx, c.workspace.DB(), query, charts)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert worldsend charts: %w", err)
	}
	return nil
}

func (c *OfficialConsolidator) lookupGenreID(catname string) int {
	if len(c.genreIDByKey) == 0 {
		return 0
	}
	key := strings.TrimSpace(strings.ToUpper(catname))
	return c.genreIDByKey[key]
}

// nullIfEmpty は空文字列の場合にnilを返します（DB挿入用）
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (c *OfficialConsolidator) loadActiveOfficialSongs(ctx context.Context) (map[string]int, error) {
	type songRecord struct {
		ID          int    `db:"id"`
		OfficialIdx string `db:"official_idx"`
	}

	var rows []songRecord
	// MySQL から既存曲データを読み込む(workspace ではなく本番データを対象にする)
	query := `SELECT id, official_idx FROM songs WHERE is_deleted = 0 AND official_idx IS NOT NULL`

	if err := c.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to load active MySQL songs: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.OfficialIdx)
		if key == "" {
			continue
		}
		result[key] = row.ID
	}

	return result, nil
}

func loadDifficultyMap(ctx context.Context, repo domainrepo.DifficultyRepository) (map[string]int, error) {
	difficulties, err := repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query difficulties: %w", err)
	}

	result := make(map[string]int)
	for _, d := range difficulties {
		result[strings.ToUpper(d.Name)] = d.ID
	}

	return result, nil
}

func defaultDifficultyMap() map[string]int {
	return map[string]int{
		"BASIC":    1,
		"ADVANCED": 2,
		"EXPERT":   3,
		"MASTER":   4,
		"ULTIMA":   5,
	}
}

func loadGenreMap(ctx context.Context, repo domainrepo.GenreRepository) (map[string]int, error) {
	genres, err := repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query genres: %w", err)
	}

	result := make(map[string]int)
	for _, g := range genres {
		result[strings.ToUpper(g.Name)] = g.ID
	}

	return result, nil
}
