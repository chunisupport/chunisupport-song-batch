package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
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

// Consolidate は NATUA ソースから BPM を補完します。
// ノーツ数の補完は st1027 データソースが担当します。
func (c *NatuaConsolidator) Consolidate(ctx context.Context) error {
	if c.data == nil || len(c.data.Songs) == 0 {
		slog.Warn("Natua data is empty; skipping consolidation")
		return nil
	}

	officialMap, err := BuildOfficialIndexMap(ctx, c.workspace.DB())
	if err != nil {
		return err
	}

	if err := c.bulkUpdateSongBPMs(ctx, officialMap); err != nil {
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
