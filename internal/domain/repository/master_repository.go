package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
)

// Difficulty は難易度マスタのエンティティです。
type Difficulty struct {
	ID   int
	Name string
}

// CourseRepository はコースマスタを永続化します。
type CourseRepository interface {
	// SaveAll はコースをまとめて登録または更新します。
	SaveAll(ctx context.Context, courses []entity.Course) error
}

// DifficultyRepository は難易度マスタのリポジトリインターフェースです。
type DifficultyRepository interface {
	// FindAll は全ての難易度を取得します。
	FindAll(ctx context.Context) ([]Difficulty, error)
}

// Genre はジャンルマスタのエンティティです。
type Genre struct {
	ID   int
	Name string
}

// GenreRepository はジャンルマスタのリポジトリインターフェースです。
type GenreRepository interface {
	// FindAll は全てのジャンルを取得します。
	FindAll(ctx context.Context) ([]Genre, error)
}
