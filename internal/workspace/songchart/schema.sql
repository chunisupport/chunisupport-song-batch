PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO genres (id, name) VALUES
    (1, 'POPS & ANIME'),
    (2, 'niconico'),
    (3, '東方Project'),
    (4, 'VARIETY'),
    (5, 'イロドリミドリ'),
    (6, 'ゲキマイ'),
    (7, 'ORIGINAL');

CREATE TABLE IF NOT EXISTS difficulties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO difficulties (id, name) VALUES
    (1, 'BASIC'),
    (2, 'ADVANCED'),
    (3, 'EXPERT'),
    (4, 'MASTER'),
    (5, 'ULTIMA');

CREATE TABLE IF NOT EXISTS songs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    display_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    reading TEXT,
    artist TEXT NOT NULL,
    genre_id INTEGER NOT NULL,
    bpm INTEGER,
    released_at TEXT,
    official_idx TEXT NOT NULL UNIQUE,
    jacket TEXT,
    is_worldsend INTEGER NOT NULL DEFAULT 0 CHECK(is_worldsend IN (0,1)),
    is_new INTEGER NOT NULL DEFAULT 0 CHECK(is_new IN (0,1)),
    is_deleted INTEGER NOT NULL DEFAULT 0 CHECK(is_deleted IN (0,1)),
    FOREIGN KEY(genre_id) REFERENCES genres(id) ON DELETE CASCADE,
    CHECK(bpm IS NULL OR bpm > 0)
);

CREATE TABLE IF NOT EXISTS charts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL,
    difficulty_id INTEGER NOT NULL,
    const REAL NOT NULL CHECK(const >= 0),
    is_const_unknown INTEGER NOT NULL DEFAULT 1 CHECK(is_const_unknown IN (0,1)),
    notes INTEGER,
    notes_designer TEXT,
    FOREIGN KEY(song_id) REFERENCES songs(id) ON DELETE CASCADE,
    FOREIGN KEY(difficulty_id) REFERENCES difficulties(id) ON DELETE CASCADE,
    UNIQUE(song_id, difficulty_id),
    CHECK(notes IS NULL OR notes >= 0)
);

CREATE TABLE IF NOT EXISTS worldsend_charts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id INTEGER NOT NULL UNIQUE,
    level_star INTEGER CHECK(level_star IS NULL OR level_star BETWEEN 1 AND 5),
    attribute TEXT CHECK(LENGTH(attribute) <= 1),
    notes INTEGER CHECK(notes IS NULL OR notes >= 0),
    notes_designer TEXT,
    FOREIGN KEY(song_id) REFERENCES songs(id) ON DELETE CASCADE
);
