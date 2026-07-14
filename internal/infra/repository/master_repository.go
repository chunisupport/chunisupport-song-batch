package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
)

// difficultyRepositoryImpl は DifficultyRepository のインフラ層実装です。
type difficultyRepositoryImpl struct {
	db DBorTx
}

type courseRepositoryImpl struct {
	db DBorTx
}

// NewCourseRepository は CourseRepository の実装を生成します。
func NewCourseRepository(db DBorTx) domainrepo.CourseRepository {
	return &courseRepositoryImpl{db: db}
}

// SaveAll はクラスを一括取得した後、コースをまとめて登録または更新します。
func (r *courseRepositoryImpl) SaveAll(ctx context.Context, courses []entity.Course) error {
	if len(courses) == 0 {
		return nil
	}

	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM course_classes`)
	if err != nil {
		return fmt.Errorf("failed to query course classes: %w", err)
	}
	defer rows.Close()

	classIDs := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("failed to scan course class: %w", err)
		}
		classIDs[strings.ToLower(strings.TrimSpace(name))] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during course classes iteration: %w", err)
	}

	const chunkSize = 500
	for start := 0; start < len(courses); start += chunkSize {
		end := min(start+chunkSize, len(courses))
		chunk := courses[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*4)
		for i, course := range chunk {
			classID, ok := classIDs[strings.ToLower(strings.TrimSpace(course.ClassName))]
			if !ok {
				return fmt.Errorf("course %q references unknown class %q", course.OfficialIdx, course.ClassName)
			}
			values[i] = "(?, ?, ?, ?, 0)"
			args = append(args, course.DisplayID.String(), course.OfficialIdx, course.Name, classID)
		}

		query := `INSERT INTO courses (display_id, official_idx, name, course_class_id, is_deleted) VALUES ` + strings.Join(values, ",") + `
ON DUPLICATE KEY UPDATE
	name = VALUES(name),
	course_class_id = VALUES(course_class_id),
	is_deleted = 0`
		if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to save courses (%d-%d): %w", start, end, err)
		}
	}

	return nil
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
