package service

import (
	"context"
	"database/sql"
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

	consolidator := NewOtogeDbConsolidator(ws, &data, "")
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

	consolidator := NewOtogeDbConsolidator(ws, &data, "")
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

func TestOtogeDbConsolidate_UpdatesWikiPageTitle(t *testing.T) {
	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	_, err = ws.DB().ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, wiki_page_title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES
			(1, 'song-1', 'Blow my mind', 'Old Title', 'Artist', 4, '100', 0, 0),
			(2, 'song-2', 'Other Wiki', NULL, 'Artist', 4, '200', 0, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert songs: %v", err)
	}

	data := importer.OtogeDbData{
		{ID: "100", Title: "Blow my mind", WikiwikiURL: "https://wikiwiki.jp/chunithmwiki/Blow%20my%20mind"},
		{ID: "200", Title: "Other Wiki", WikiwikiURL: "https://example.com/Other Wiki"},
	}

	consolidator := NewOtogeDbConsolidator(ws, &data, "https://wikiwiki.jp/chunithmwiki/")
	if err := consolidator.Consolidate(ctx); err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	var titles []sql.NullString
	if err := ws.DB().SelectContext(ctx, &titles, `SELECT wiki_page_title FROM songs ORDER BY id`); err != nil {
		t.Fatalf("failed to get wiki_page_title: %v", err)
	}
	if !titles[0].Valid || titles[0].String != "Blow my mind" {
		t.Errorf("expected wiki_page_title=Blow my mind, got %v", titles[0])
	}
	if titles[1].Valid {
		t.Errorf("expected wiki_page_title to remain NULL for non-matching base URL, got %v", titles[1].String)
	}
}

func TestExtractWikiPageTitle(t *testing.T) {
	t.Parallel()

	const baseURL = "https://wikiwiki.jp/chunithmwiki/"
	tests := []struct {
		name   string
		url    string
		want   string
		wantOK bool
	}{
		{name: "エンコードなし", url: baseURL + "可愛くてごめん", want: "可愛くてごめん", wantOK: true},
		{name: "一部エンコード", url: baseURL + "トウキョウ・シャンディ・ランデヴ feat. 花譜%2C ツミキ", want: "トウキョウ・シャンディ・ランデヴ feat. 花譜, ツミキ", wantOK: true},
		{name: "全体エンコード", url: baseURL + "%E3%83%88%E3%83%AA%E3%82%B9%E3%83%A1%E3%82%AE%E3%82%B9%E3%83%88%E3%82%B9(%E6%A5%BD%E6%9B%B2%E5%90%8D)", want: "トリスメギストス(楽曲名)", wantOK: true},
		{name: "不正なエスケープは元の文字列を使う", url: baseURL + "100%", want: "100%", wantOK: true},
		{name: "空文字列", url: "", wantOK: false},
		{name: "ベースURLのみ", url: baseURL, wantOK: false},
		{name: "ベースURLが異なる", url: "https://example.com/ALIVE", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := extractWikiPageTitle(tt.url, baseURL)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("extractWikiPageTitle(%q) = (%q, %v), want (%q, %v)", tt.url, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
