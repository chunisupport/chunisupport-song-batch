package service

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

func TestSt1027BulkUpdateChartNotes_InitialInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES (1, 1, 10.5, 0, NULL), (1, 2, 12.0, 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
				},
				Basic:    importer.St1027Chart{NotesAll: ptr(500)},
				Advanced: importer.St1027Chart{NotesAll: ptr(800)},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateChartNotes(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateChartNotes returned error: %v", err)
	}

	var notes1, notes2 *int
	err = ws.DB().GetContext(ctx, &notes1, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 1`)
	if err != nil {
		t.Fatalf("failed to get notes for chart 1: %v", err)
	}
	err = ws.DB().GetContext(ctx, &notes2, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 2`)
	if err != nil {
		t.Fatalf("failed to get notes for chart 2: %v", err)
	}

	if notes1 == nil || *notes1 != 500 {
		t.Errorf("expected notes1=500, got %v", notes1)
	}
	if notes2 == nil || *notes2 != 800 {
		t.Errorf("expected notes2=800, got %v", notes2)
	}
}

func TestSt1027BulkUpdateChartNotes_NoUpdateExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES (1, 1, 10.5, 0, 400), (1, 2, 12.0, 0, 700)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
				},
				Basic:    importer.St1027Chart{NotesAll: ptr(500)},
				Advanced: importer.St1027Chart{NotesAll: ptr(800)},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateChartNotes(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateChartNotes returned error: %v", err)
	}

	var notes1, notes2 *int
	err = ws.DB().GetContext(ctx, &notes1, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 1`)
	if err != nil {
		t.Fatalf("failed to get notes for chart 1: %v", err)
	}
	err = ws.DB().GetContext(ctx, &notes2, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 2`)
	if err != nil {
		t.Fatalf("failed to get notes for chart 2: %v", err)
	}

	if notes1 == nil || *notes1 != 400 {
		t.Errorf("expected notes1=400 (not updated from 400), got %v", notes1)
	}
	if notes2 == nil || *notes2 != 700 {
		t.Errorf("expected notes2=700 (not updated from 700), got %v", notes2)
	}
}

func TestSt1027BulkUpdateChartNotes_NoOverwriteWithZero(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES (1, 1, 10.5, 0, 500)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
				},
				Basic: importer.St1027Chart{NotesAll: ptr(0)},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateChartNotes(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateChartNotes returned error: %v", err)
	}

	var notes *int
	err = ws.DB().GetContext(ctx, &notes, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 1`)
	if err != nil {
		t.Fatalf("failed to get notes for chart: %v", err)
	}

	if notes == nil || *notes != 500 {
		t.Errorf("expected notes to remain 500 (not overwritten by 0), got %v", notes)
	}
}

func TestSt1027BulkUpdateSongBPMs_InitialInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
				},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm=180, got %v", bpm)
	}
}

func TestSt1027BulkUpdateSongBPMs_NoOverwriteExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, 160)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
				},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 160 {
		t.Errorf("expected bpm to remain 160 (not overwritten), got %v", bpm)
	}
}

func TestSt1027BulkUpdateSongBPMs_NoOverwriteWithZero(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, 180)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
					BPM:        ptr(0),
				},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm to remain 180 (not overwritten by 0), got %v", bpm)
	}
}

func TestSt1027Consolidator_Consolidate_FullFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer)
		VALUES (1, 3, 12.5, 0, NULL, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test chart: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
				},
				Expert: importer.St1027Chart{
					NotesAll:      ptr(900),
					Notesdesigner: ptrString("Techno Kitchen"),
				},
			},
		},
	}

	consolidator := NewSt1027Consolidator(ws, st1027Data)
	if err := consolidator.Consolidate(ctx); err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}
	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm=180, got %v", bpm)
	}

	var notes *int
	err = ws.DB().GetContext(ctx, &notes, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 3`)
	if err != nil {
		t.Fatalf("failed to get notes for chart: %v", err)
	}
	if notes == nil || *notes != 900 {
		t.Errorf("expected notes=900, got %v", notes)
	}

	var designer *string
	err = ws.DB().GetContext(ctx, &designer, `SELECT notes_designer FROM charts WHERE song_id = 1 AND difficulty_id = 3`)
	if err != nil {
		t.Fatalf("failed to get notes_designer for chart: %v", err)
	}
	if designer == nil || *designer != "Techno Kitchen" {
		t.Errorf("expected notes_designer=Techno Kitchen, got %v", designer)
	}
}

func TestSt1027BulkUpdateChartNotesDesigner_InitialInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer)
		VALUES (1, 3, 12.5, 0, 900, NULL), (1, 4, 13.0, 0, 1000, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
				},
				Expert: importer.St1027Chart{Notesdesigner: ptrString("Techno Kitchen")},
				Master: importer.St1027Chart{Notesdesigner: ptrString("Jack")},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateChartNotesDesigner(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateChartNotesDesigner returned error: %v", err)
	}

	var designer1, designer2 *string
	err = ws.DB().GetContext(ctx, &designer1, `SELECT notes_designer FROM charts WHERE song_id = 1 AND difficulty_id = 3`)
	if err != nil {
		t.Fatalf("failed to get notes_designer for chart 1: %v", err)
	}
	err = ws.DB().GetContext(ctx, &designer2, `SELECT notes_designer FROM charts WHERE song_id = 1 AND difficulty_id = 4`)
	if err != nil {
		t.Fatalf("failed to get notes_designer for chart 2: %v", err)
	}

	if designer1 == nil || *designer1 != "Techno Kitchen" {
		t.Errorf("expected designer1=Techno Kitchen, got %v", designer1)
	}
	if designer2 == nil || *designer2 != "Jack" {
		t.Errorf("expected designer2=Jack, got %v", designer2)
	}
}

func TestSt1027BulkUpdateChartNotesDesigner_NoUpdateExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer)
		VALUES (1, 3, 12.5, 0, 900, 'Existing Designer')
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	st1027Data := &importer.St1027Data{
		Songs: []importer.St1027Song{
			{
				Meta: importer.St1027Meta{
					OfficialID: "OFF001",
				},
				Expert: importer.St1027Chart{Notesdesigner: ptrString("New Designer")},
			},
		},
	}

	consolidator := &St1027Consolidator{
		workspace: ws,
		data:      st1027Data,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateChartNotesDesigner(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateChartNotesDesigner returned error: %v", err)
	}

	var designer *string
	err = ws.DB().GetContext(ctx, &designer, `SELECT notes_designer FROM charts WHERE song_id = 1 AND difficulty_id = 3`)
	if err != nil {
		t.Fatalf("failed to get notes_designer for chart: %v", err)
	}

	if designer == nil || *designer != "Existing Designer" {
		t.Errorf("expected notes_designer to remain Existing Designer, got %v", designer)
	}
}

func ptrString(v string) *string {
	return &v
}
