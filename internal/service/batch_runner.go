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
	newCourseRepo  func(domainrepo.ExtendedDBExecutor) domainrepo.CourseRepository
	pwPepper       string
	wikiBaseURL    string
}

// NewBatchRunner は BatchRunner を生成します。
func NewBatchRunner(
	db domainrepo.ExtendedDBExecutor,
	tm domainrepo.TransactionManager,
	difficultyRepo domainrepo.DifficultyRepository,
	genreRepo domainrepo.GenreRepository,
	newCourseRepo func(domainrepo.ExtendedDBExecutor) domainrepo.CourseRepository,
	pwPepper string,
	wikiBaseURL string,
) *BatchRunner {
	return &BatchRunner{
		db:             db,
		tm:             tm,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		newCourseRepo:  newCourseRepo,
		pwPepper:       pwPepper,
		wikiBaseURL:    wikiBaseURL,
	}
}

// Consolidate は必須条件を満たしたソースをワークスペース経由で同期します。
func (r *BatchRunner) Consolidate(ctx context.Context, sources ConsolidationSources, names []string, opts ConsolidationOptions) error {
	svc := NewConsolidationService(r.db, r.difficultyRepo, r.genreRepo, nil, r.pwPepper, r.wikiBaseURL, names, opts, sources)
	workspace, err := svc.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return fmt.Errorf("no datasources to consolidate")
	}
	defer workspace.Close()

	return r.tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		return svc.SyncWorkspace(ctx, workspace, tx, r.newCourseRepo(tx))
	})
}
