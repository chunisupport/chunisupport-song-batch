package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	domainrepo "chunisupport-song-batch/internal/domain/repository"
	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/util"
	"chunisupport-song-batch/internal/workspace/songchart"

	"github.com/jmoiron/sqlx"
)

type difficultyInfo struct {
	ID   int
	Name string
}

type songRecordForUpsert struct {
	DisplayID   string `db:"display_id"`
	Title       string `db:"title"`
	Artist      string `db:"artist"`
	GenreID     int    `db:"genre_id"`
	OfficialIdx string `db:"official_idx"`
	Jacket      any    `db:"jacket"`
	IsWorldsEnd int    `db:"is_worldsend"`
}

type chartRecordForUpsert struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
}

type worldsendChartRecordForUpsert struct {
	SongID  int     `db:"song_id"`
	WeStar  *int    `db:"we_star"`
	WeKanji *string `db:"we_kanji"`
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

func (c *OfficialConsolidator) prepareSongsForUpsert() ([]songRecordForUpsert, map[string]struct{}) {
	var songsToUpsert []songRecordForUpsert
	seenOfficialIdx := make(map[string]struct{})

	for _, song := range *c.data {
		officialID := strings.TrimSpace(song.ID)
		if officialID == "" {
			slog.Warn("Skipping official song without official_idx", "title", song.Title, "artist", song.Artist)
			continue
		}

		genreID := c.lookupGenreID(song.Catname)
		if genreID == 0 {
			slog.Warn("Skipping official song with unknown genre", "title", song.Title, "catname", song.Catname)
			continue
		}

		isWorldsEnd := strings.TrimSpace(song.WeKanji) != "" || strings.TrimSpace(song.WeStar) != ""
		image := normalizeImage(strings.TrimSpace(song.Image))
		displayID := GenerateDisplayID()

		songsToUpsert = append(songsToUpsert, songRecordForUpsert{
			DisplayID:   displayID,
			Title:       strings.TrimSpace(song.Title),
			Artist:      strings.TrimSpace(song.Artist),
			GenreID:     genreID,
			OfficialIdx: officialID,
			Jacket:      nullIfEmpty(image),
			IsWorldsEnd: util.BoolToInt(isWorldsEnd),
		})
		seenOfficialIdx[officialID] = struct{}{}
	}
	return songsToUpsert, seenOfficialIdx
}

func (c *OfficialConsolidator) bulkUpsertSongs(ctx context.Context, songs []songRecordForUpsert) error {
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

func (c *OfficialConsolidator) prepareChartsForUpsert(songIDs map[string]int) []chartRecordForUpsert {
	var chartsToUpsert []chartRecordForUpsert

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

		isWorldsEnd := strings.TrimSpace(song.WeKanji) != "" || strings.TrimSpace(song.WeStar) != ""
		if isWorldsEnd {
			continue
		}

		for diffName, getLevel := range levelByDifficultyKey {
			level := strings.TrimSpace(getLevel(&song))
			if level == "" {
				continue
			}

			diffID := c.difficultyMap[diffName]
			if diffID == 0 {
				slog.Warn("Unknown difficulty in workspace; skipping official chart", "difficulty", diffName, "song", song.Title)
				continue
			}

			constValue, err := parseOfficialLevel(level)
			if err != nil {
				slog.Warn("Failed to parse official level", "level", level, "difficulty", diffName, "error", err)
				continue
			}

			chartsToUpsert = append(chartsToUpsert, chartRecordForUpsert{
				SongID:         songID,
				DifficultyID:   diffID,
				Const:          constValue,
				IsConstUnknown: util.BoolToInt(constValue >= 10.0),
			})
		}
	}
	return chartsToUpsert
}

func (c *OfficialConsolidator) bulkUpsertCharts(ctx context.Context, charts []chartRecordForUpsert) error {
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

func (c *OfficialConsolidator) prepareWorldsendChartsForUpsert(songIDs map[string]int) []worldsendChartRecordForUpsert {
	var chartsToUpsert []worldsendChartRecordForUpsert

	for _, song := range *c.data {
		officialID := strings.TrimSpace(song.ID)
		songID, ok := songIDs[officialID]
		if !ok {
			continue
		}

		isWorldsEnd := strings.TrimSpace(song.WeKanji) != "" || strings.TrimSpace(song.WeStar) != ""
		if !isWorldsEnd {
			continue
		}

		weKanji := strings.TrimSpace(song.WeKanji)
		weStar := strings.TrimSpace(song.WeStar)
		var weStarInt *int
		if weStar != "" {
			starValue, err := strconv.Atoi(weStar)
			if err != nil {
				slog.Warn("Failed to parse we_star as integer", "we_star", weStar, "song", song.Title, "error", err)
			} else {
				mappedStar := mapWeStarToActualCount(starValue)
				if mappedStar == 0 {
					slog.Warn("Invalid we_star value", "we_star", starValue, "song", song.Title, "expected", "1, 3, 5, 7, or 9")
				} else {
					weStarInt = &mappedStar
				}
			}
		}

		var weKanjiStr *string
		if weKanji != "" {
			runes := []rune(weKanji)
			if len(runes) > 1 {
				slog.Warn("we_kanji longer than 1 character, truncating", "we_kanji", weKanji, "song", song.Title)
				truncated := string(runes[:1])
				weKanjiStr = &truncated
			} else {
				weKanjiStr = &weKanji
			}
		}

		chartsToUpsert = append(chartsToUpsert, worldsendChartRecordForUpsert{
			SongID:  songID,
			WeStar:  weStarInt,
			WeKanji: weKanjiStr,
		})
	}

	return chartsToUpsert
}

func (c *OfficialConsolidator) bulkUpsertWorldsendCharts(ctx context.Context, charts []worldsendChartRecordForUpsert) error {
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

func parseOfficialLevel(level string) (float64, error) {
	level = strings.ReplaceAll(level, "+", ".5")
	return strconv.ParseFloat(level, 64)
}

func normalizeImage(image string) string {
	if image == "" {
		return ""
	}
	if dotIndex := strings.LastIndex(image, "."); dotIndex != -1 {
		return image[:dotIndex]
	}
	return image
}

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

// mapWeStarToActualCount は公式データの星表記を実際の星の数に変換します
// 1→1個, 3→2個, 5→3個, 7→4個, 9→5個
func mapWeStarToActualCount(officialValue int) int {
	switch officialValue {
	case 1:
		return 1
	case 3:
		return 2
	case 5:
		return 3
	case 7:
		return 4
	case 9:
		return 5
	default:
		return 0 // 不正な値
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
