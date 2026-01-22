package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/workspace/songchart"
)

// OtogeDbConsolidator は otoge-db 由来のリリース日を補完します。
type OtogeDbConsolidator struct {
	workspace *songchart.SongChartWorkspace
	data      *importer.OtogeDbData
}

// NewOtogeDbConsolidator は OtogeDbConsolidator の新しいインスタンスを作成します。
func NewOtogeDbConsolidator(workspace *songchart.SongChartWorkspace, data *importer.OtogeDbData) *OtogeDbConsolidator {
	return &OtogeDbConsolidator{
		workspace: workspace,
		data:      data,
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
