package service

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// TestSt1027BulkUpdateChartNotes_InitialInsert は notes が NULL の場合に正しく更新されることを確認
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

	// テスト用の楽曲と譜面を挿入
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

	// st1027 データを準備
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

	// 検証
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

// TestSt1027BulkUpdateChartNotes_NoUpdateExisting は既存の notes が上書きされないことを確認
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

	// テスト用の楽曲と譜面を挿入（既に notes が設定されている）
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

	// st1027 データ: 異なる notes 値を持つ
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

	// 検証: 更新されていないべき
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

// TestSt1027BulkUpdateChartNotes_NoOverwriteWithZero は 0 や null で上書きしないことを確認
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

	// テスト用の楽曲と譜面を挿入（既に正の notes が設定されている）
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

	// st1027 データ: notes_all が 0 と null（上書きしないべき）
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

	// 検証: 500 のまま維持されているべき
	var notes *int
	err = ws.DB().GetContext(ctx, &notes, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 1`)
	if err != nil {
		t.Fatalf("failed to get notes for chart: %v", err)
	}

	if notes == nil || *notes != 500 {
		t.Errorf("expected notes to remain 500 (not overwritten by 0), got %v", notes)
	}
}
