package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// MainframeConsolidator は mainframe 由来の定数を補完します。
type MainframeConsolidator struct {
	workspace *songchart.SongChartWorkspace
	data      *importer.MainframeData
}

// NewMainframeConsolidator は MainframeConsolidator の新しいインスタンスを作成します。
func NewMainframeConsolidator(workspace *songchart.SongChartWorkspace, data *importer.MainframeData) *MainframeConsolidator {
	return &MainframeConsolidator{
		workspace: workspace,
		data:      data,
	}
}

// Consolidate は mainframe の譜面情報をワークスペースへ反映します。
func (c *MainframeConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(*c.data) == 0 {
		slog.Warn("Mainframe data is empty; skipping consolidation")
		return nil
	}

	chartMap, err := c.buildChartMap(ctx)
	if err != nil {
		return err
	}

	var updated int
	for _, chart := range *c.data {
		title := strings.TrimSpace(chart.Title)
		genre := strings.TrimSpace(chart.Genre)
		if title == "" || chart.Const <= 0 {
			continue
		}

		diffID := difficulty.ParseName(chart.Diff).Int()
		if diffID == 0 {
			continue
		}

		// タイトル、ジャンル、難易度を正規化してマッチングキーを作成
		normalizedTitle := normalizer(title)
		normalizedGenre := normalizer(genre)
		matchKey := fmt.Sprintf("%s|%s|%d", normalizedTitle, normalizedGenre, diffID)

		songID, exists := chartMap[matchKey]
		if !exists {
			continue
		}

		result, err := c.workspace.DB().ExecContext(ctx, `
UPDATE charts
SET const = ?, is_const_unknown = 0
WHERE song_id = ? AND difficulty_id = ?
`, chart.Const, songID, diffID)
		if err != nil {
			return fmt.Errorf("failed to update mainframe chart song_id=%d diff=%d: %w", songID, diffID, err)
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			updated++
		}
	}

	slog.Info("Mainframe chart constants updated", "count", updated)
	return nil
}

func (c *MainframeConsolidator) buildChartMap(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		SongID       int    `db:"song_id"`
		DifficultyID int    `db:"difficulty_id"`
		Title        string `db:"title"`
		GenreName    string `db:"genre_name"`
	}
	query := `
		SELECT c.song_id, c.difficulty_id, s.title, COALESCE(g.name, '') as genre_name
		FROM charts c
		INNER JOIN songs s ON c.song_id = s.id
		LEFT JOIN genres g ON s.genre_id = g.id
		WHERE s.is_deleted = 0
	`
	if err := c.workspace.DB().SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to build workspace chart map: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, row := range rows {
		// タイトルとジャンルを正規化
		normalizedTitle := normalizer(row.Title)
		normalizedGenre := normalizer(row.GenreName)

		// タイトル+ジャンル+難易度のキー
		key := fmt.Sprintf("%s|%s|%d", normalizedTitle, normalizedGenre, row.DifficultyID)
		result[key] = row.SongID
	}
	return result, nil
}

// normalizer はタイトル文字列を正規化します。
// NFKC正規化を行い、特殊文字を統一し、空白・制御文字・フォーマット文字を削除し、小文字に変換します。
func normalizer(title string) string {
	// JSONエスケープシーケンスを除去
	title = strings.ReplaceAll(title, `\"`, `"`) // \" → "
	title = strings.ReplaceAll(title, `\'`, `'`) // \' → '

	// NFKC正規化（全角英数字を半角に、全角カナを半角に統一）
	// これにより全角の１→1、ＡＢＣ→ABCなどが変換される
	normalized := norm.NFKC.String(title)

	// 引用符と波線を統一 (NFKC正規化後に実施)
	normalized = strings.ReplaceAll(normalized, "\u201C", `"`) // " (LEFT DOUBLE QUOTATION MARK) → "
	normalized = strings.ReplaceAll(normalized, "\u201D", `"`) // " (RIGHT DOUBLE QUOTATION MARK) → "
	normalized = strings.ReplaceAll(normalized, "\u2018", `'`) // ' (LEFT SINGLE QUOTATION MARK) → '
	normalized = strings.ReplaceAll(normalized, "\u2019", `'`) // ' (RIGHT SINGLE QUOTATION MARK) → '
	normalized = strings.ReplaceAll(normalized, "\u301C", `~`) // 〜 (WAVE DASH) → ~
	normalized = strings.ReplaceAll(normalized, "\uFF5E", `~`) // ～ (FULLWIDTH TILDE) → ~

	// 空白・制御文字・フォーマット文字を削除
	var builder strings.Builder
	for _, r := range normalized {
		// 空白、制御文字、フォーマット文字（U+202A等）を除外
		if !unicode.IsSpace(r) && !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r) {
			builder.WriteRune(r)
		}
	}

	// 小文字に変換
	return strings.ToLower(builder.String())
}
