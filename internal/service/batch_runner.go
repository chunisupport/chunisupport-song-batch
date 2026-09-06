package service

import (
	"context"
	"fmt"

	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
)

// BatchRunner は統合と MySQL 同期を実行します。
type BatchRunner struct {
	db             domainrepo.ExtendedDBExecutor
	tm             domainrepo.TransactionManager
	difficultyRepo domainrepo.DifficultyRepository
	genreRepo      domainrepo.GenreRepository
	courseRepo     domainrepo.CourseRepository
	pwPepper       string
}

// NewBatchRunner は BatchRunner を生成します。
func NewBatchRunner(
	db domainrepo.ExtendedDBExecutor,
	tm domainrepo.TransactionManager,
	difficultyRepo domainrepo.DifficultyRepository,
	genreRepo domainrepo.GenreRepository,
	courseRepo domainrepo.CourseRepository,
	pwPepper string,
) *BatchRunner {
	return &BatchRunner{
		db:             db,
		tm:             tm,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		courseRepo:     courseRepo,
		pwPepper:       pwPepper,
	}
}

// Consolidate は必須条件を満たしたソースをワークスペース経由で同期します。
func (r *BatchRunner) Consolidate(ctx context.Context, sources ConsolidationSources, names []string, opts ConsolidationOptions) error {
	svc := NewConsolidationService(r.db, r.difficultyRepo, r.genreRepo, r.courseRepo, r.pwPepper, names, opts, sources)
	workspace, err := svc.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return fmt.Errorf("no datasources to consolidate")
	}
	defer workspace.Close()

	return r.tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		return svc.SyncWorkspace(ctx, workspace, tx)
	})
}
