package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// maxWikiPageTitleLength は MySQL の songs.wiki_page_title (VARCHAR(300)) に格納できる最大文字数です。
const maxWikiPageTitleLength = 300

// OtogeDbConsolidator は otoge-db 由来のリリース日やWikiページタイトルなどを補完します。
type OtogeDbConsolidator struct {
	workspace   *songchart.SongChartWorkspace
	data        *importer.OtogeDbData
	wikiBaseURL string
}

// NewOtogeDbConsolidator は OtogeDbConsolidator の新しいインスタンスを作成します。
// wikiBaseURL が空の場合、Wikiページタイトルの補完は行いません。
func NewOtogeDbConsolidator(workspace *songchart.SongChartWorkspace, data *importer.OtogeDbData, wikiBaseURL string) *OtogeDbConsolidator {
	return &OtogeDbConsolidator{
		workspace:   workspace,
		data:        data,
		wikiBaseURL: wikiBaseURL,
	}
}

// Consolidate は otoge-db の楽曲情報をワークスペースへ反映します。
// otoge-dbのIDとofficial_idxでマッチングしてリリース日を更新します。
func (c *OtogeDbConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(*c.data) == 0 {
		slog.Warn("Otoge-db data is empty; skipping consolidation")
		return nil
	}

	// official_idxでマッチングするためのマップを構築
	idxMap, err := c.buildIdxMap(ctx)
	if err != nil {
		return err
	}

	if err := c.bulkUpdateSongBPMs(ctx, idxMap); err != nil {
		return err
	}
	if err := c.bulkUpdateWorldsendChartNotes(ctx, idxMap); err != nil {
		return err
	}
	if err := c.bulkUpdateWorldsendChartNotesDesigner(ctx, idxMap); err != nil {
		return err
	}
	if err := c.bulkUpdateSongWikiPageTitles(ctx, idxMap); err != nil {
		return err
	}

	var updated int
	for _, song := range *c.data {
		dateAdded := strings.TrimSpace(song.DateAdded)
		idStr := strings.TrimSpace(song.ID)

		if dateAdded == "" || idStr == "" || len(dateAdded) != 8 {
			continue
		}

		// IDを文字列から整数に変換
		id, err := strconv.Atoi(idStr)
		if err != nil {
			slog.Debug("Failed to parse otoge-db ID", "id", idStr, "title", song.Title, "error", err)
			continue
		}

		if id <= 0 {
			continue
		}

		// YYYYMMDD形式をYYYY-MM-DD形式に変換
		releaseDate := fmt.Sprintf("%s-%s-%s", dateAdded[0:4], dateAdded[4:6], dateAdded[6:8])

		// IDで楽曲を検索
		songID, exists := idxMap[id]
		if !exists {
			continue
		}

		// リリース日を更新（既存のリリース日がNULLの場合のみ）
		result, err := c.workspace.DB().ExecContext(ctx, `
UPDATE songs
SET released_at = ?
WHERE id = ? AND released_at IS NULL
`, releaseDate, songID)
		if err != nil {
			return fmt.Errorf("failed to update otoge-db song id=%d: %w", songID, err)
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			updated++
		}
	}

	slog.Info("Otoge-db release dates updated", "count", updated)
	return nil
}

func (c *OtogeDbConsolidator) bulkUpdateSongBPMs(ctx context.Context, idxMap map[int]int) error {
	var records []SongBPMRecord
	for _, song := range *c.data {
		if !hasWorldsEndMetadata(song) {
			continue
		}

		songID, bpm, ok := c.extractSongIDAndPositiveInt(song.ID, song.BPM, idxMap)
		if !ok {
			continue
		}

		records = append(records, SongBPMRecord{
			ID:  songID,
			BPM: bpm,
		})
	}

	if len(records) == 0 {
		return nil
	}

	affected, err := ExecuteBulkUpdateSongBPMs(ctx, c.workspace.DB(), records)
	if err != nil {
		return fmt.Errorf("failed to bulk update otoge-db song bpms: %w", err)
	}

	slog.Info("Otoge-db songs bpm updated", "count", affected)
	return nil
}

func (c *OtogeDbConsolidator) bulkUpdateWorldsendChartNotes(ctx context.Context, idxMap map[int]int) error {
	var records []WorldsendChartNotesRecord
	for _, song := range *c.data {
		songID, notes, ok := c.extractSongIDAndPositiveInt(song.ID, song.LevWENotes, idxMap)
		if !ok {
			continue
		}

		records = append(records, WorldsendChartNotesRecord{
			SongID: songID,
			Notes:  notes,
		})
	}

	if len(records) == 0 {
		return nil
	}

	affected, err := BulkUpdateWorldsendChartNotesInBatches(ctx, c.workspace.DB(), records)
	if err != nil {
		return fmt.Errorf("failed to bulk update otoge-db WORLD'S END chart notes: %w", err)
	}

	slog.Info("Otoge-db WORLD'S END chart notes updated", "count", affected)
	return nil
}

func (c *OtogeDbConsolidator) bulkUpdateWorldsendChartNotesDesigner(ctx context.Context, idxMap map[int]int) error {
	var records []WorldsendChartNotesDesignerRecord
	for _, song := range *c.data {
		songID, ok := c.extractSongID(song.ID, idxMap)
		if !ok {
			continue
		}

		notesDesigner := strings.TrimSpace(song.LevWEDesigner)
		if notesDesigner == "" {
			continue
		}

		records = append(records, WorldsendChartNotesDesignerRecord{
			SongID:        songID,
			NotesDesigner: notesDesigner,
		})
	}

	if len(records) == 0 {
		return nil
	}

	affected, err := BulkUpdateWorldsendChartNotesDesignerInBatches(ctx, c.workspace.DB(), records)
	if err != nil {
		return fmt.Errorf("failed to bulk update otoge-db WORLD'S END chart notes_designer: %w", err)
	}

	slog.Info("Otoge-db WORLD'S END chart notes_designer updated", "count", affected)
	return nil
}

func (c *OtogeDbConsolidator) bulkUpdateSongWikiPageTitles(ctx context.Context, idxMap map[int]int) error {
	if c.wikiBaseURL == "" {
		slog.Warn("Wiki base URL is not configured; skipping wiki_page_title consolidation")
		return nil
	}

	var records []SongWikiPageTitleRecord
	for _, song := range *c.data {
		songID, ok := c.extractSongID(song.ID, idxMap)
		if !ok {
			continue
		}

		wikiPageTitle, ok := extractWikiPageTitle(song.WikiwikiURL, c.wikiBaseURL)
		if !ok {
			continue
		}
		if utf8.RuneCountInString(wikiPageTitle) > maxWikiPageTitleLength {
			slog.Warn("Otoge-db wiki page title is too long; skipping", "id", song.ID, "title", song.Title)
			continue
		}

		records = append(records, SongWikiPageTitleRecord{
			ID:            songID,
			WikiPageTitle: wikiPageTitle,
		})
	}

	if len(records) == 0 {
		return nil
	}

	affected, err := BulkUpdateSongWikiPageTitlesInBatches(ctx, c.workspace.DB(), records)
	if err != nil {
		return fmt.Errorf("failed to bulk update otoge-db song wiki_page_title: %w", err)
	}

	slog.Info("Otoge-db songs wiki_page_title updated", "count", affected)
	return nil
}

// extractWikiPageTitle は wikiwiki_url からベースURLを取り除き、Wikiのページタイトルを取り出します。
// otoge-db の wikiwiki_url はパーセントエンコードの有無が楽曲ごとに混在しているため、
// デコードしてタイトルの表記を統一します。デコードできない場合は元の文字列をそのまま使います。
func extractWikiPageTitle(wikiwikiURL, wikiBaseURL string) (string, bool) {
	title, found := strings.CutPrefix(strings.TrimSpace(wikiwikiURL), wikiBaseURL)
	if !found {
		return "", false
	}

	if decoded, err := url.PathUnescape(title); err == nil {
		title = decoded
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return "", false
	}
	return title, true
}

func (c *OtogeDbConsolidator) extractSongID(idStr string, idxMap map[int]int) (int, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(idStr))
	if err != nil || id <= 0 {
		return 0, false
	}

	songID, exists := idxMap[id]
	if !exists {
		return 0, false
	}

	return songID, true
}

func (c *OtogeDbConsolidator) extractSongIDAndPositiveInt(idStr, value string, idxMap map[int]int) (int, int, bool) {
	songID, ok := c.extractSongID(idStr, idxMap)
	if !ok {
		return 0, 0, false
	}

	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return 0, 0, false
	}

	return songID, n, true
}

func hasWorldsEndMetadata(song importer.OtogeDbSong) bool {
	return strings.TrimSpace(song.WeKanji) != "" ||
		strings.TrimSpace(song.WeStar) != "" ||
		strings.TrimSpace(song.LevWENotes) != "" ||
		strings.TrimSpace(song.LevWEDesigner) != ""
}

func (c *OtogeDbConsolidator) buildIdxMap(ctx context.Context) (map[int]int, error) {
	var rows []struct {
		ID          int `db:"id"`
		OfficialIdx int `db:"official_idx"`
	}
	if err := c.workspace.DB().SelectContext(ctx, &rows, `SELECT id, official_idx FROM songs WHERE is_deleted = 0 AND official_idx IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("failed to build workspace idx map: %w", err)
	}

	result := make(map[int]int, len(rows))
	for _, row := range rows {
		result[row.OfficialIdx] = row.ID
	}
	return result, nil
}
