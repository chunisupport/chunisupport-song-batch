package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

const (
	difficultyBasic    = "basic"
	difficultyAdvanced = "advanced"
	difficultyExpert   = "expert"
	difficultyMaster   = "master"
	difficultyUltima   = "ultima"
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
	if err := c.bulkUpdateChartNotesDesigner(ctx, officialMap); err != nil {
		return err
	}
	if err := c.bulkUpdateSongBPMs(ctx, officialMap); err != nil {
		return err
	}

	slog.Info("St1027 data consolidation completed")
	return nil
}
func (c *St1027Consolidator) bulkUpdateChartNotes(ctx context.Context, officialMap map[string]int) error {
	var records []ChartNotesRecord
	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}

		chartsToProcess := map[string]importer.St1027Chart{
			difficultyBasic:    song.Basic,
			difficultyAdvanced: song.Advanced,
			difficultyExpert:   song.Expert,
			difficultyMaster:   song.Master,
			difficultyUltima:   song.Ultima,
		}

		for name, chart := range chartsToProcess {
			diffID := difficulty.ParseName(name).Int()
			if diffID == 0 {
				continue
			}
			if chart.NotesAll == nil || *chart.NotesAll <= 0 {
				continue
			}
			records = append(records, ChartNotesRecord{
				SongID:       songID,
				DifficultyID: diffID,
				Notes:        *chart.NotesAll,
			})
		}
	}

	if len(records) == 0 {
		return nil
	}

	totalAffected, err := BulkUpdateChartNotesInBatches(ctx, c.workspace.DB(), records)
	if err != nil {
		return err
	}

	slog.Info("St1027 chart notes updated", "count", totalAffected)
	return nil
}

func (c *St1027Consolidator) bulkUpdateChartNotesDesigner(ctx context.Context, officialMap map[string]int) error {
	var records []ChartNotesDesignerRecord
	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}

		chartsToProcess := map[string]importer.St1027Chart{
			difficultyBasic:    song.Basic,
			difficultyAdvanced: song.Advanced,
			difficultyExpert:   song.Expert,
			difficultyMaster:   song.Master,
			difficultyUltima:   song.Ultima,
		}

		for name, chart := range chartsToProcess {
			diffID := difficulty.ParseName(name).Int()
			if diffID == 0 || chart.Notesdesigner == nil {
				continue
			}

			notesDesigner := strings.TrimSpace(*chart.Notesdesigner)
			if notesDesigner == "" {
				continue
			}

			records = append(records, ChartNotesDesignerRecord{
				SongID:        songID,
				DifficultyID:  diffID,
				NotesDesigner: notesDesigner,
			})
		}
	}

	if len(records) == 0 {
		return nil
	}

	totalAffected, err := BulkUpdateChartNotesDesignerInBatches(ctx, c.workspace.DB(), records)
	if err != nil {
		return err
	}

	slog.Info("St1027 chart notes_designer updated", "count", totalAffected)
	return nil
}

func (c *St1027Consolidator) bulkUpdateSongBPMs(ctx context.Context, officialMap map[string]int) error {
	var records []SongBPMRecord
	for _, song := range c.data.Songs {
		officialID := strings.TrimSpace(song.Meta.OfficialID)
		if officialID == "" {
			continue
		}
		songID, exists := officialMap[officialID]
		if !exists {
			continue
		}
		if song.Meta.BPM == nil || *song.Meta.BPM <= 0 {
			continue
		}

		records = append(records, SongBPMRecord{ID: songID, BPM: *song.Meta.BPM})
	}

	if len(records) == 0 {
		return nil
	}

	affected, err := ExecuteBulkUpdateSongBPMs(ctx, c.workspace.DB(), records)
	if err != nil {
		return err
	}

	slog.Info("St1027 songs bpm updated", "count", affected)
	return nil
}
