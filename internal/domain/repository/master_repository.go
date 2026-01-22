package repository

import (
	"context"
)

// Difficulty は難易度マスタのエンティティです。
type Difficulty struct {
	ID   int
	Name string
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
