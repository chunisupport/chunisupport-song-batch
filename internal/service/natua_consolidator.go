package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"chunisupport-song-batch/internal/domain/difficulty"
	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/info"
	"chunisupport-song-batch/internal/workspace/songchart"
)

type songBPMRecord struct {
	ID  int `db:"id"`
	BPM int `db:"bpm"`
}

type chartNotesRecord struct {
	SongID       int `db:"song_id"`
	DifficultyID int `db:"difficulty_id"`
	Notes        int `db:"notes"`
}

// bulkUpdateSongBPMs 用のテンプレート（パフォーマンスのため事前にパースしておく）
// 差分検知: 既存の BPM を 0 や null で上書きしないようにする
var bulkUpdateSongBpmsTpl = template.Must(template.New("bulkUpdateSongBPMs").Parse(`
UPDATE songs SET bpm = CASE id
	{{- range .}}
	WHEN {{.ID}} THEN {{.BPM}}
	{{- end}}
END
WHERE id IN (
	{{- range $i, $e := .}}
	{{- if $i}},{{end}}{{.ID}}
	{{- end -}}
) AND (bpm IS NULL OR bpm = 0)
`))

// bulkUpdateChartNotes 用のテンプレート（パフォーマンスのため事前にパースしておく）
// SQLiteでは UPDATE ... FROM 構文が使えないため、CASE を使う
// 差分検知: 既存の notes を 0 や null で上書きしないようにする
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

// NatuaConsolidator は NATUA データの補完を担当します。
type NatuaConsolidator struct {
	workspace *songchart.SongChartWorkspace
	data      *importer.NatuaData
}

// NewNatuaConsolidator は NatuaConsolidator を初期化します。
func NewNatuaConsolidator(workspace *songchart.SongChartWorkspace, data *importer.NatuaData) *NatuaConsolidator {
	return &NatuaConsolidator{
		workspace: workspace,
		data:      data,
	}
}

// Consolidate は NATUA ソースから BPM や NOTES を補完します。
func (c *NatuaConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(c.data.Songs) == 0 {
		slog.Warn("Natua data is empty; skipping consolidation")
		return nil
	}

	officialMap, err := c.buildOfficialIndexMap(ctx)
	if err != nil {
		return err
	}

	if err := c.bulkUpdateSongBPMs(ctx, officialMap); err != nil {
		return err
	}
	if err := c.bulkUpdateChartNotes(ctx, officialMap); err != nil {
		return err
	}

	slog.Info("Natua data consolidation completed")
	return nil
}

func (c *NatuaConsolidator) bulkUpdateSongBPMs(ctx context.Context, officialMap map[string]int) error {
	var records []songBPMRecord
	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}
		if song.Meta.BPM != nil && !song.Meta.BPMNodata && *song.Meta.BPM > 0 {
			records = append(records, songBPMRecord{ID: songID, BPM: *song.Meta.BPM})
		}
	}

	if len(records) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := bulkUpdateSongBpmsTpl.Execute(&buf, records); err != nil {
		return fmt.Errorf("failed to execute bulk update song bpms template: %w", err)
	}

	result, err := c.workspace.DB().ExecContext(ctx, buf.String())
	if err != nil {
		return fmt.Errorf("failed to execute bulk update song bpms: %w", err)
	}

	affected, _ := result.RowsAffected()
	slog.Info("Natua songs metadata updated", "count", affected)
	return nil
}

func (c *NatuaConsolidator) bulkUpdateChartNotes(ctx context.Context, officialMap map[string]int) error {
	var records []chartNotesRecord
	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}

		for diff, chart := range map[string]importer.NatuaChart{
			"basic":    song.Basic,
			"advanced": song.Advanced,
			"expert":   song.Expert,
			"master":   song.Master,
			"ultima":   song.Ultima,
		} {
			diffID := difficulty.ParseName(diff).Int()
			if diffID == 0 {
				continue
			}
			if chart.Notes == nil || *chart.Notes < 0 || chart.NotesNodata {
				continue
			}
			records = append(records, chartNotesRecord{
				SongID:       songID,
				DifficultyID: diffID,
				Notes:        *chart.Notes,
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

	slog.Info("Natua chart notes updated", "count", totalAffected)
	return nil
}

func (c *NatuaConsolidator) executeBulkUpdateChartNotes(ctx context.Context, records []chartNotesRecord) (int64, error) {
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

func (c *NatuaConsolidator) buildOfficialIndexMap(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		ID          int    `db:"id"`
		OfficialIdx string `db:"official_idx"`
	}
	if err := c.workspace.DB().SelectContext(ctx, &rows, `SELECT id, official_idx FROM songs WHERE official_idx IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("failed to build workspace official_idx map: %w", err)
	}

	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.OfficialIdx] = row.ID
	}
	return result, nil
}
