package service

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/difficulty"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// TestBulkUpdateSongBPMs_InitialInsert は BPM が NULL の場合に正しく更新されることを確認
func TestBulkUpdateSongBPMs_InitialInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テスト用の楽曲を挿入（BPM は NULL）
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	// NATUA データを準備
	natuaData := &importer.NatuaData{
		Songs: []importer.NatuaSong{
			{
				Meta: importer.NatuaMeta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
					BPMNodata:  false,
				},
			},
		},
	}

	consolidator := &NatuaConsolidator{
		workspace: ws,
		data:      natuaData,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	// 検証
	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm=180, got %v", bpm)
	}
}

// TestBulkUpdateSongBPMs_NoOverwriteExisting は既存の BPM が上書きされないことを確認
func TestBulkUpdateSongBPMs_NoOverwriteExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テスト用の楽曲を挿入（既に BPM が設定されている）
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, 160)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	// NATUA データ: 異なる BPM 値を持つ
	natuaData := &importer.NatuaData{
		Songs: []importer.NatuaSong{
			{
				Meta: importer.NatuaMeta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
					BPMNodata:  false,
				},
			},
		},
	}

	consolidator := &NatuaConsolidator{
		workspace: ws,
		data:      natuaData,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	// 検証: 既存の BPM (160) がそのまま維持されるべき（上書きしない仕様）
	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 160 {
		t.Errorf("expected bpm to remain 160 (not overwritten), got %v", bpm)
	}
}

// TestBulkUpdateSongBPMs_NoOverwriteWithZero は 0 で上書きしないことを確認
func TestBulkUpdateSongBPMs_NoOverwriteWithZero(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テスト用の楽曲を挿入（既に正の BPM が設定されている）
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, 180)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	// NATUA データ: 0 を含む（上書きしないべき）
	natuaData := &importer.NatuaData{
		Songs: []importer.NatuaSong{
			{
				Meta: importer.NatuaMeta{
					OfficialID: "OFF001",
					BPM:        ptr(0),
					BPMNodata:  false,
				},
			},
		},
	}

	consolidator := &NatuaConsolidator{
		workspace: ws,
		data:      natuaData,
	}

	officialMap := map[string]int{"OFF001": 1}
	err = consolidator.bulkUpdateSongBPMs(ctx, officialMap)
	if err != nil {
		t.Fatalf("bulkUpdateSongBPMs returned error: %v", err)
	}

	// 検証: 180 のまま維持されているべき
	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm for song: %v", err)
	}

	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm to remain 180 (not overwritten by 0), got %v", bpm)
	}
}

// TestNatuaConsolidator_Consolidate_EmptyData は空データの場合にエラーにならないことを確認
func TestNatuaConsolidator_Consolidate_EmptyData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// 空データ
	emptyData := &importer.NatuaData{
		Songs: []importer.NatuaSong{},
	}

	consolidator := NewNatuaConsolidator(ws, emptyData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate with empty data should not return error: %v", err)
	}
}

// TestNatuaConsolidator_Consolidate_NilData は nil データの場合にエラーにならないことを確認
func TestNatuaConsolidator_Consolidate_NilData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	consolidator := NewNatuaConsolidator(ws, nil)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate with nil data should not return error: %v", err)
	}
}

// TestNatuaConsolidator_Consolidate_FullFlow は BPM のみが更新されることを確認
// ノーツ数の補完は st1027 データソースが担当するため、natua では更新しない
func TestNatuaConsolidator_Consolidate_FullFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テストデータ挿入
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES 
			(1, 1, 5.0, 0, NULL),
			(1, 2, 8.0, 0, NULL),
			(1, 3, 10.5, 0, NULL),
			(1, 4, 12.5, 0, NULL),
			(1, 5, 14.0, 0, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	// NATUA データを準備
	natuaData := &importer.NatuaData{
		Songs: []importer.NatuaSong{
			{
				Meta: importer.NatuaMeta{
					OfficialID: "OFF001",
					BPM:        ptr(180),
					BPMNodata:  false,
				},
				Basic:    importer.NatuaChart{Notes: ptr(300)},
				Advanced: importer.NatuaChart{Notes: ptr(500)},
				Expert:   importer.NatuaChart{Notes: ptr(800)},
				Master:   importer.NatuaChart{Notes: ptr(1200)},
				Ultima:   importer.NatuaChart{Notes: ptr(1500)},
			},
		},
	}

	consolidator := NewNatuaConsolidator(ws, natuaData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	// BPM の検証
	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm: %v", err)
	}
	if bpm == nil || *bpm != 180 {
		t.Errorf("expected bpm=180, got %v", bpm)
	}

	// Notes は更新されないべき（st1027 が担当）
	var noteRows []struct {
		Notes *int `db:"notes"`
	}
	err = ws.DB().SelectContext(ctx, &noteRows, `SELECT notes FROM charts WHERE song_id = 1 ORDER BY difficulty_id`)
	if err != nil {
		t.Fatalf("failed to get notes: %v", err)
	}
	for i, row := range noteRows {
		if row.Notes != nil {
			t.Errorf("difficulty %d: expected notes=NULL, got %d", i+1, *row.Notes)
		}
	}
}

// TestNatuaConsolidator_Consolidate_SkipsNodata は nodata フラグがある場合にスキップすることを確認
func TestNatuaConsolidator_Consolidate_SkipsNodata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テストデータ挿入（既に BPM と Notes が設定されている）
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted, bpm)
		VALUES (1, 'disp-001', 'Test Song', 'Test Artist', 1, 'OFF001', 0, 150)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES (1, 1, 5.0, 0, 400)
	`)
	if err != nil {
		t.Fatalf("failed to insert test chart: %v", err)
	}

	// nodata フラグが設定されている NATUA データ
	natuaData := &importer.NatuaData{
		Songs: []importer.NatuaSong{
			{
				Meta: importer.NatuaMeta{
					OfficialID: "OFF001",
					BPM:        ptr(200),
					BPMNodata:  true, // nodata フラグ
				},
				Basic: importer.NatuaChart{
					Notes:       ptr(500),
					NotesNodata: true, // nodata フラグ
				},
			},
		},
	}

	consolidator := NewNatuaConsolidator(ws, natuaData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	// BPM は変更されないべき（nodata）
	var bpm *int
	err = ws.DB().GetContext(ctx, &bpm, `SELECT bpm FROM songs WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to get bpm: %v", err)
	}
	if bpm == nil || *bpm != 150 {
		t.Errorf("expected bpm to remain 150, got %v", bpm)
	}

	// Notes は変更されないべき（nodata）
	var notes *int
	err = ws.DB().GetContext(ctx, &notes, `SELECT notes FROM charts WHERE song_id = 1 AND difficulty_id = 1`)
	if err != nil {
		t.Fatalf("failed to get notes: %v", err)
	}
	if notes == nil || *notes != 400 {
		t.Errorf("expected notes to remain 400, got %v", notes)
	}
}

// TestBuildOfficialIndexMap は official_idx マップが正しく構築されることを確認
// 注: BuildOfficialIndexMap は is_deleted をフィルタしない（全ての楽曲を返す）
func TestBuildOfficialIndexMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テストデータ挿入
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES 
			(1, 'disp-001', 'Song 1', 'Artist 1', 1, 'OFF001', 0),
			(2, 'disp-002', 'Song 2', 'Artist 2', 1, 'OFF002', 0),
			(3, 'disp-003', 'Song 3', 'Artist 3', 1, 'OFF003', 1)
	`)
	if err != nil {
		t.Fatalf("failed to insert test songs: %v", err)
	}

	idxMap, err := BuildOfficialIndexMap(ctx, ws.DB())
	if err != nil {
		t.Fatalf("BuildOfficialIndexMap returned error: %v", err)
	}

	// 検証: official_idx が NOT NULL のすべての楽曲がマップに含まれる（is_deleted は考慮しない）
	if len(idxMap) != 3 {
		t.Errorf("expected 3 entries in idxMap, got %d", len(idxMap))
	}

	if id, ok := idxMap["OFF001"]; !ok || id != 1 {
		t.Errorf("expected idxMap['OFF001']=1, got %d, exists=%v", id, ok)
	}
	if id, ok := idxMap["OFF002"]; !ok || id != 2 {
		t.Errorf("expected idxMap['OFF002']=2, got %d, exists=%v", id, ok)
	}
	if id, ok := idxMap["OFF003"]; !ok || id != 3 {
		t.Errorf("expected idxMap['OFF003']=3, got %d, exists=%v", id, ok)
	}
}

// TestMapDifficultyTypeToID は難易度マッピングを確認
func TestMapDifficultyTypeToID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected int
	}{
		{"basic", 1},
		{"advanced", 2},
		{"expert", 3},
		{"master", 4},
		{"ultima", 5},
		{"worldsend", 6},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := difficulty.ParseName(tt.input).Int()
			if result != tt.expected {
				t.Errorf("difficulty.ParseName(%q).Int() = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}
