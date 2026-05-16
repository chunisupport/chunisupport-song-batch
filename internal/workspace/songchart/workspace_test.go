package songchart

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestChartKey はチャートキーの生成を確認
func TestChartKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		songID       int
		difficultyID int
		expected     string
	}{
		{1, 1, "1:1"},
		{100, 5, "100:5"},
		{0, 0, "0:0"},
	}

	for _, tt := range tests {
		result := chartKey(tt.songID, tt.difficultyID)
		if result != tt.expected {
			t.Errorf("chartKey(%d, %d) = %s, expected %s", tt.songID, tt.difficultyID, result, tt.expected)
		}
	}
}

// TestNullableInt は sql.NullInt64 から適切な値を返すことを確認
func TestNullableInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    sql.NullInt64
		expected any
	}{
		{"valid value", sql.NullInt64{Int64: 100, Valid: true}, int64(100)},
		{"null value", sql.NullInt64{Valid: false}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullableInt(tt.value)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestNullableString は sql.NullString から適切な値を返すことを確認
func TestNullableString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    sql.NullString
		expected any
	}{
		{"valid value", sql.NullString{String: "test", Valid: true}, "test"},
		{"null value", sql.NullString{Valid: false}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullableString(tt.value)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestParseSQLStatements は SQL 文を正しく分割することを確認
func TestParseSQLStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single statement",
			input:    "CREATE TABLE test (id INT);",
			expected: []string{"CREATE TABLE test (id INT)"},
		},
		{
			name:     "multiple statements",
			input:    "CREATE TABLE a (id INT); CREATE TABLE b (id INT);",
			expected: []string{"CREATE TABLE a (id INT)", "CREATE TABLE b (id INT)"},
		},
		{
			name:     "with whitespace",
			input:    "  SELECT 1;  \n  SELECT 2;  ",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "empty statements ignored",
			input:    "SELECT 1;; ; SELECT 2;",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseSQLStatements(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d statements, got %d", len(tt.expected), len(result))
			}
			for i, stmt := range result {
				if stmt != tt.expected[i] {
					t.Errorf("statement %d: expected %q, got %q", i, tt.expected[i], stmt)
				}
			}
		})
	}
}

// TestNewSongChartWorkspace はワークスペースの作成を確認
func TestNewSongChartWorkspace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := NewSongChartWorkspace(ctx, Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// スキーマが正しく作成されていることを確認
	var count int

	// genres テーブルが存在することを確認
	err = ws.DB().GetContext(ctx, &count, "SELECT COUNT(*) FROM genres")
	if err != nil {
		t.Fatalf("genres table not created: %v", err)
	}
	if count == 0 {
		t.Errorf("expected genres to be populated")
	}

	// difficulties テーブルが存在することを確認
	err = ws.DB().GetContext(ctx, &count, "SELECT COUNT(*) FROM difficulties")
	if err != nil {
		t.Fatalf("difficulties table not created: %v", err)
	}
	if count == 0 {
		t.Errorf("expected difficulties to be populated")
	}

	// songs テーブルが存在することを確認
	_, err = ws.DB().ExecContext(ctx, "SELECT * FROM songs LIMIT 1")
	if err != nil {
		t.Fatalf("songs table not created: %v", err)
	}

	// charts テーブルが存在することを確認
	_, err = ws.DB().ExecContext(ctx, "SELECT * FROM charts LIMIT 1")
	if err != nil {
		t.Fatalf("charts table not created: %v", err)
	}

	// worldsend_charts テーブルが存在することを確認
	_, err = ws.DB().ExecContext(ctx, "SELECT * FROM worldsend_charts LIMIT 1")
	if err != nil {
		t.Fatalf("worldsend_charts table not created: %v", err)
	}
}

// TestLoadWorkspaceSongs はワークスペースからの楽曲読み込みを確認
func TestLoadWorkspaceSongs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := NewSongChartWorkspace(ctx, Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テストデータ挿入
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (display_id, title, reading, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES 
			('disp-001', 'Song 1', 'そんぐ1', 'Artist 1', 1, 'OFF001', 0, 0),
			('disp-002', 'Song 2', NULL, 'Artist 2', 2, 'OFF002', 0, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test songs: %v", err)
	}

	songs, err := ws.loadWorkspaceSongs(ctx)
	if err != nil {
		t.Fatalf("loadWorkspaceSongs returned error: %v", err)
	}

	if len(songs) != 2 {
		t.Errorf("expected 2 songs, got %d", len(songs))
	}

	// 順序を確認（ORDER BY id）
	if songs[0].Title != "Song 1" || songs[1].Title != "Song 2" {
		t.Errorf("songs not in expected order")
	}
	if !songs[0].Reading.Valid || songs[0].Reading.String != "そんぐ1" {
		t.Errorf("expected reading 'そんぐ1', got %+v", songs[0].Reading)
	}
	if songs[1].Reading.Valid {
		t.Errorf("expected null reading, got %+v", songs[1].Reading)
	}
}

// TestLoadWorkspaceCharts はワークスペースからの譜面読み込みを確認
func TestLoadWorkspaceCharts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := NewSongChartWorkspace(ctx, Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// テストデータ挿入
	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES (1, 'disp-001', 'Song 1', 'Artist 1', 1, 'OFF001', 0, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test song: %v", err)
	}

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES 
			(1, 1, 5.0, 0, 300),
			(1, 2, 8.0, 0, 500),
			(1, 3, 10.5, 1, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert test charts: %v", err)
	}

	charts, err := ws.loadWorkspaceCharts(ctx)
	if err != nil {
		t.Fatalf("loadWorkspaceCharts returned error: %v", err)
	}

	if len(charts) != 3 {
		t.Errorf("expected 3 charts, got %d", len(charts))
	}
}

func TestResolveChartUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		existing        mysqlChart
		exists          bool
		chart           workspaceChart
		opts            SyncOptions
		expectedAction  syncAction
		expectedConst   float64
		expectedUnknown bool
		expectedNotes   sql.NullInt64
	}{
		// 挿入ケース
		{
			name:            "Insert Standard",
			exists:          false,
			chart:           workspaceChart{Const: 10.5, IsConstUnknown: true, Notes: sql.NullInt64{Valid: true, Int64: 500}},
			opts:            SyncOptions{},
			expectedAction:  actionInsert,
			expectedConst:   10.5,
			expectedUnknown: true,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 500},
		},
		{
			name:            "Insert MajorUpdate Low Level",
			exists:          false,
			chart:           workspaceChart{Const: 5.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: true, Int64: 300}},
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionInsert,
			expectedConst:   5.0,
			expectedUnknown: false, // 既知に強制
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 300},
		},
		{
			name:            "Insert MajorUpdate High Level",
			exists:          false,
			chart:           workspaceChart{Const: 12.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionInsert,
			expectedConst:   12.0,
			expectedUnknown: true, // 不明に強制
			expectedNotes:   sql.NullInt64{Valid: false},
		},

		// 標準更新ケース
		{
			name:            "Standard Const Mismatch",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 400}},
			exists:          true,
			chart:           workspaceChart{Const: 10.5, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{},
			expectedAction:  actionSkip,
			expectedConst:   10.0, // 既存値を維持
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 400}, // 既存ノーツを維持
		},
		{
			name:            "Standard Const Match Notes Update",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			exists:          true,
			chart:           workspaceChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 100}},
			opts:            SyncOptions{},
			expectedAction:  actionUpdate,
			expectedConst:   10.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 100},
		},
		{
			name:            "Standard Unknown to Known (Value Change)",
			existing:        mysqlChart{Const: 14.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: true, Int64: 600}},
			exists:          true,
			chart:           workspaceChart{Const: 14.2, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{},
			expectedAction:  actionUpdate,
			expectedConst:   14.2,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 600}, // 既存ノーツを維持
		},
		{
			name:            "Standard Unknown to Known",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: false}},
			exists:          true,
			chart:           workspaceChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 200}},
			opts:            SyncOptions{},
			expectedAction:  actionUpdate,
			expectedConst:   10.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 200},
		},
		{
			name:            "Standard Known to Unknown",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 350}},
			exists:          true,
			chart:           workspaceChart{Const: 10.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{},
			expectedAction:  actionSkip, // ブロック
			expectedConst:   10.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 350}, // 既存ノーツを維持
		},

		// MajorUpdate ケース
		{
			name:            "MajorUpdate Low Level Force Update",
			existing:        mysqlChart{Const: 5.5, IsConstUnknown: true, Notes: sql.NullInt64{Valid: true, Int64: 450}},
			exists:          true,
			chart:           workspaceChart{Const: 5.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}}, // 公式値 5.0
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionUpdate,
			expectedConst:   5.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 450}, // 既存ノーツを維持
		},
		{
			name:            "MajorUpdate High Level No Contradiction",
			existing:        mysqlChart{Const: 14.2, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 700}},
			exists:          true,
			chart:           workspaceChart{Const: 14.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: false}}, // レベル14
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionUpdate, // Unknown=true に設定するため更新
			expectedConst:   14.2,         // 既存値を維持
			expectedUnknown: true,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 700}, // 既存ノーツを維持
		},
		{
			name:            "MajorUpdate High Level Contradiction (Low)",
			existing:        mysqlChart{Const: 13.9, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 800}},
			exists:          true,
			chart:           workspaceChart{Const: 14.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: false}}, // レベル14
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionUpdate,
			expectedConst:   14.0, // 上書き
			expectedUnknown: true,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 800}, // 既存ノーツを維持
		},
		{
			name:            "MajorUpdate High Level Contradiction (High)",
			existing:        mysqlChart{Const: 14.6, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 900}},
			exists:          true,
			chart:           workspaceChart{Const: 14.0, IsConstUnknown: true, Notes: sql.NullInt64{Valid: false}}, // レベル14
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionUpdate,
			expectedConst:   14.0, // 上書き
			expectedUnknown: true,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 900}, // 既存ノーツを維持
		},
		// ノーツ数のNULL上書き防止テスト
		{
			name:            "Prevent NULL Overwrite - Existing Notes",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 500}},
			exists:          true,
			chart:           workspaceChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{},
			expectedAction:  actionSkip,
			expectedConst:   10.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 500}, // 既存ノーツを保持
		},
		{
			name:            "Allow Notes Update - NULL to Value",
			existing:        mysqlChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			exists:          true,
			chart:           workspaceChart{Const: 10.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 250}},
			opts:            SyncOptions{},
			expectedAction:  actionUpdate,
			expectedConst:   10.0,
			expectedUnknown: false,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 250}, // 新しいノーツを設定
		},
		{
			name:            "MajorUpdate Prevent NULL Overwrite",
			existing:        mysqlChart{Const: 12.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: true, Int64: 650}},
			exists:          true,
			chart:           workspaceChart{Const: 12.0, IsConstUnknown: false, Notes: sql.NullInt64{Valid: false}},
			opts:            SyncOptions{MajorUpdate: true},
			expectedAction:  actionUpdate, // Unknown=trueに変更されるため更新
			expectedConst:   12.0,
			expectedUnknown: true,
			expectedNotes:   sql.NullInt64{Valid: true, Int64: 650}, // 既存ノーツを保持
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, u, n, _, a := resolveChartUpdate(tt.existing, tt.exists, tt.chart, tt.opts)
			if c != tt.expectedConst {
				t.Errorf("const mismatch: got %v, want %v", c, tt.expectedConst)
			}
			if u != tt.expectedUnknown {
				t.Errorf("unknown mismatch: got %v, want %v", u, tt.expectedUnknown)
			}
			if n.Valid != tt.expectedNotes.Valid || (n.Valid && n.Int64 != tt.expectedNotes.Int64) {
				t.Errorf("notes mismatch: got %+v, want %+v", n, tt.expectedNotes)
			}
			if a != tt.expectedAction {
				t.Errorf("action mismatch: got %v, want %v", a, tt.expectedAction)
			}
		})
	}
}

// TestBuildBulkUpdateChartsSQL は生成 SQL の構造を確認します。
// データ値はすべてプレースホルダー(?) であり、SQL 文字列にリテラル値が埋め込まれないことを保証します。
func TestBuildBulkUpdateChartsSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		n              int
		wantWhenCount  int // 各 CASE ブロックあたりの WHEN 節の数
		wantWhereCount int // WHERE IN の (?,?) の数
	}{
		{"1件", 1, 1, 1},
		{"3件", 3, 3, 3},
		{"10件", 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sql := buildBulkUpdateChartsSQL(tt.n)

			// UPDATE 文の対象テーブルを確認
			if !strings.Contains(sql, "UPDATE charts") {
				t.Error("UPDATE charts が含まれていません")
			}

			// 4列分の CASE ブロックが存在することを確認
			for _, col := range []string{"const", "is_const_unknown", "notes", "notes_designer"} {
				if !strings.Contains(sql, col+" = CASE") {
					t.Errorf("列 %s の CASE ブロックが含まれていません", col)
				}
			}

			// WHEN 節の数を確認 (4列 × n件)
			gotWhen := strings.Count(sql, "WHEN song_id = ? AND difficulty_id = ?")
			wantWhen := 4 * tt.wantWhenCount
			if gotWhen != wantWhen {
				t.Errorf("WHEN 節の数: got %d, want %d", gotWhen, wantWhen)
			}

			// WHERE IN の (?,?) の数を確認
			gotWhere := strings.Count(sql, "(?,?)")
			if gotWhere != tt.wantWhereCount {
				t.Errorf("WHERE IN の (?,?) の数: got %d, want %d", gotWhere, tt.wantWhereCount)
			}
		})
	}
}

// TestBuildBulkUpdateWorldsendChartsSQL は生成 SQL の構造を確認します。
// データ値はすべてプレースホルダー(?) であり、SQL 文字列にリテラル値が埋め込まれないことを保証します。
func TestBuildBulkUpdateWorldsendChartsSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		n              int
		wantWhenCount  int // 各 CASE ブロックあたりの WHEN 節の数
		wantWhereCount int // WHERE IN の ? の数
	}{
		{"1件", 1, 1, 1},
		{"3件", 3, 3, 3},
		{"10件", 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sql := buildBulkUpdateWorldsendChartsSQL(tt.n)

			// UPDATE 文の対象テーブルを確認
			if !strings.Contains(sql, "UPDATE worldsend_charts") {
				t.Error("UPDATE worldsend_charts が含まれていません")
			}

			// 4列分の CASE ブロックが存在することを確認
			for _, col := range []string{"level_star", "attribute", "notes", "notes_designer"} {
				if !strings.Contains(sql, col+" = CASE") {
					t.Errorf("列 %s の CASE ブロックが含まれていません", col)
				}
			}

			// WHEN 節の数を確認 (4列 × n件)
			gotWhen := strings.Count(sql, "WHEN song_id = ?")
			wantWhen := 4 * tt.wantWhenCount
			if gotWhen != wantWhen {
				t.Errorf("WHEN 節の数: got %d, want %d", gotWhen, wantWhen)
			}

			// COALESCE の数を確認 (4列 × n件)
			gotCoalesce := strings.Count(sql, "COALESCE(?")
			if gotCoalesce != wantWhen {
				t.Errorf("COALESCE の数: got %d, want %d", gotCoalesce, wantWhen)
			}

			// WHERE IN の ? の数を確認
			whereStart := strings.Index(sql, "WHERE song_id IN (")
			if whereStart == -1 {
				t.Fatal("WHERE song_id IN が含まれていません")
			}
			gotWhere := strings.Count(sql[whereStart:], "?")
			if gotWhere != tt.wantWhereCount {
				t.Errorf("WHERE IN の ? の数: got %d, want %d", gotWhere, tt.wantWhereCount)
			}
		})
	}
}
