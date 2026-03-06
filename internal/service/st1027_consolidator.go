package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// St1027Consolidator は st1027 データからノーツ数を補完します。
type St1027Consolidator struct {
	workspace *songchart.SongChartWorkspace
	data      *importer.St1027Data
}

// NewSt1027Consolidator は St1027Consolidator を初期化します。
func NewSt1027Consolidator(workspace *songchart.SongChartWorkspace, data *importer.St1027Data) *St1027Consolidator {
	return &St1027Consolidator{
		workspace: workspace,
		data:      data,
	}
}

// Consolidate は st1027 ソースからノーツ数を補完します。
func (c *St1027Consolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(c.data.Songs) == 0 {
		slog.Warn("St1027 data is empty; skipping consolidation")
		return nil
	}

	officialMap, err := BuildOfficialIndexMap(ctx, c.workspace.DB())
	if err != nil {
		return err
	}

	if err := c.bulkUpdateChartNotes(ctx, officialMap); err != nil {
		return err
	}

	slog.Info("St1027 data consolidation completed")
	return nil
}

// st1027の難易度フィールド名から difficulty.ParseName で使用する名称へのマッピング
var st1027DiffMap = map[string]func(s *importer.St1027Song) *importer.St1027Chart{
	"basic":    func(s *importer.St1027Song) *importer.St1027Chart { return &s.Basic },
	"advanced": func(s *importer.St1027Song) *importer.St1027Chart { return &s.Advanced },
	"expert":   func(s *importer.St1027Song) *importer.St1027Chart { return &s.Expert },
	"master":   func(s *importer.St1027Song) *importer.St1027Chart { return &s.Master },
	"ultima":   func(s *importer.St1027Song) *importer.St1027Chart { return &s.Ultima },
}

func (c *St1027Consolidator) bulkUpdateChartNotes(ctx context.Context, officialMap map[string]int) error {
	var records []chartNotesRecord
	for i := range c.data.Songs {
		song := &c.data.Songs[i]
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}

		for diffName, getChart := range st1027DiffMap {
			diffID := difficulty.ParseName(diffName).Int()
			if diffID == 0 {
				continue
			}
			chart := getChart(song)
			if chart.NotesAll == nil || *chart.NotesAll <= 0 {
				continue
			}
			records = append(records, chartNotesRecord{
				SongID:       songID,
				DifficultyID: diffID,
				Notes:        *chart.NotesAll,
			})
		}
	}

	if len(records) == 0 {
		return nil
	}

	// バッチに分割して処理（SQLiteのUNION ALL制限対策）
	var totalAffected int64
	for i := 0; i < len(records); i += info.SQLiteCompoundSelectLimit {
		end := min(i+info.SQLiteCompoundSelectLimit, len(records))
		batch := records[i:end]

		affected, err := c.executeBulkUpdateChartNotes(ctx, batch)
		if err != nil {
			return err
		}
		totalAffected += affected
	}

	slog.Info("St1027 chart notes updated", "count", totalAffected)
	return nil
}

func (c *St1027Consolidator) executeBulkUpdateChartNotes(ctx context.Context, records []chartNotesRecord) (int64, error) {
	var buf bytes.Buffer
	if err := bulkUpdateChartNotesTpl.Execute(&buf, records); err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes template: %w", err)
	}

	result, err := c.workspace.DB().ExecContext(ctx, buf.String())
	if err != nil {
		return 0, fmt.Errorf("failed to execute bulk update chart notes: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}
