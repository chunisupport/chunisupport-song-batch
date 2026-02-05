package repository

import (
	"context"
	"fmt"
	"strings"

	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
)

// difficultyRepositoryImpl は DifficultyRepository のインフラ層実装です。
type difficultyRepositoryImpl struct {
	db DBorTx
}

// NewDifficultyRepository は DifficultyRepository の実装を生成します。
func NewDifficultyRepository(db DBorTx) domainrepo.DifficultyRepository {
	return &difficultyRepositoryImpl{db: db}
}

// FindAll は全ての難易度を取得します。
func (r *difficultyRepositoryImpl) FindAll(ctx context.Context) ([]domainrepo.Difficulty, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM difficulties`)
	if err != nil {
		return nil, fmt.Errorf("failed to query difficulties: %w", err)
	}
	defer rows.Close()

	var result []domainrepo.Difficulty
	for rows.Next() {
		var d domainrepo.Difficulty
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, fmt.Errorf("failed to scan difficulty: %w", err)
		}
		d.Name = strings.TrimSpace(d.Name)
		result = append(result, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during difficulties iteration: %w", err)
	}

	return result, nil
}

// genreRepositoryImpl は GenreRepository のインフラ層実装です。
type genreRepositoryImpl struct {
	db DBorTx
}

// NewGenreRepository は GenreRepository の実装を生成します。
func NewGenreRepository(db DBorTx) domainrepo.GenreRepository {
	return &genreRepositoryImpl{db: db}
}

// FindAll は全てのジャンルを取得します。
func (r *genreRepositoryImpl) FindAll(ctx context.Context) ([]domainrepo.Genre, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM genres`)
	if err != nil {
		return nil, fmt.Errorf("failed to query genres: %w", err)
	}
	defer rows.Close()

	var result []domainrepo.Genre
	for rows.Next() {
		var g domainrepo.Genre
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("failed to scan genre: %w", err)
		}
		g.Name = strings.TrimSpace(g.Name)
		result = append(result, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during genres iteration: %w", err)
	}

	return result, nil
}
