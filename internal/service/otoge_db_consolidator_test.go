package service

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

func TestOtogeDbConsolidate_ComplementsWorldsEndData(t *testing.T) {
	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted, bpm)
		VALUES (1, 'we-random', 'Random', 'Sobrem × Silentroom', 4, '8244', 1, 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer)
		VALUES (1, 5, '分', NULL, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert worldsend chart: %v", err)
	}

	data := importer.OtogeDbData{
		{
			ID:            "8244",
			Title:         "Random",
			BPM:           "132",
			LevWENotes:    "1563",
			LevWEDesigner: "Techno Kitchen + ?",
			DateAdded:     "20221013",
		},
	}

	consolidator := NewOtogeDbConsolidator(ws, &data)
	if err := consolidator.Consolidate(ctx); err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	var bpm *int
	if err := ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`); err != nil {
		t.Fatalf("failed to get bpm: %v", err)
	}
	if bpm == nil || *bpm != 132 {
		t.Errorf("expected bpm=132, got %v", bpm)
	}

	var notes *int
	if err := ws.DB().GetContext(ctx, &notes, `SELECT notes FROM worldsend_charts WHERE song_id = 1`); err != nil {
		t.Fatalf("failed to get worldsend notes: %v", err)
	}
	if notes == nil || *notes != 1563 {
		t.Errorf("expected notes=1563, got %v", notes)
	}

	var designer *string
	if err := ws.DB().GetContext(ctx, &designer, `SELECT notes_designer FROM worldsend_charts WHERE song_id = 1`); err != nil {
		t.Fatalf("failed to get worldsend notes_designer: %v", err)
	}
	if designer == nil || *designer != "Techno Kitchen + ?" {
		t.Errorf("expected notes_designer=Techno Kitchen + ?, got %v", designer)
	}

	var releasedAt *string
	if err := ws.DB().GetContext(ctx, &releasedAt, `SELECT released_at FROM songs WHERE id = 1`); err != nil {
		t.Fatalf("failed to get released_at: %v", err)
	}
	if releasedAt == nil || *releasedAt != "2022-10-13" {
		t.Errorf("expected released_at=2022-10-13, got %v", releasedAt)
	}
}

func TestOtogeDbConsolidate_DoesNotOverwriteExistingWorldsEndData(t *testing.T) {
	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted, bpm, released_at)
		VALUES (1, 'we-random', 'Random', 'Sobrem × Silentroom', 4, '8244', 1, 0, 150, '2022-10-01')
	`)
	if err != nil {
		t.Fatalf("failed to insert song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer)
		VALUES (1, 5, '分', 999, 'Existing Designer')
	`)
	if err != nil {
		t.Fatalf("failed to insert worldsend chart: %v", err)
	}

	data := importer.OtogeDbData{
		{
			ID:            "8244",
			Title:         "Random",
			BPM:           "132",
			LevWENotes:    "1563",
			LevWEDesigner: "Techno Kitchen + ?",
			DateAdded:     "20221013",
		},
	}

	consolidator := NewOtogeDbConsolidator(ws, &data)
	if err := consolidator.Consolidate(ctx); err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	var bpm *int
	if err := ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`); err != nil {
		t.Fatalf("failed to get bpm: %v", err)
	}
	if bpm == nil || *bpm != 150 {
		t.Errorf("expected bpm to remain 150, got %v", bpm)
	}

	var notes *int
	if err := ws.DB().GetContext(ctx, &notes, `SELECT notes FROM worldsend_charts WHERE song_id = 1`); err != nil {
		t.Fatalf("failed to get worldsend notes: %v", err)
	}
	if notes == nil || *notes != 999 {
		t.Errorf("expected notes to remain 999, got %v", notes)
	}

	var designer *string
	if err := ws.DB().GetContext(ctx, &designer, `SELECT notes_designer FROM worldsend_charts WHERE song_id = 1`); err != nil {
		t.Fatalf("failed to get worldsend notes_designer: %v", err)
	}
	if designer == nil || *designer != "Existing Designer" {
		t.Errorf("expected notes_designer to remain Existing Designer, got %v", designer)
	}

	var releasedAt *string
	if err := ws.DB().GetContext(ctx, &releasedAt, `SELECT released_at FROM songs WHERE id = 1`); err != nil {
		t.Fatalf("failed to get released_at: %v", err)
	}
	if releasedAt == nil || *releasedAt != "2022-10-01" {
		t.Errorf("expected released_at to remain 2022-10-01, got %v", releasedAt)
	}
}
