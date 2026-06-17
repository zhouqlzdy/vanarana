package store

import (
	"context"
	"strings"

	"vanarana/internal/model"

	"gorm.io/gorm/clause"
)

func (s *Store) UpsertRepository(ctx context.Context, repoURL string) (*model.Repository, error) {
	name := repoURL
	if idx := strings.LastIndex(repoURL, "/"); idx >= 0 {
		name = repoURL[idx+1:]
	}
	name = strings.TrimSuffix(name, ".git")

	repo := &model.Repository{RepoURL: repoURL, Name: name}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "repo_url"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(repo).Error
	if err != nil {
		return nil, err
	}
	return s.GetRepositoryByURL(ctx, repoURL)
}

func (s *Store) GetRepositoryByURL(ctx context.Context, repoURL string) (*model.Repository, error) {
	var r model.Repository
	err := s.db.WithContext(ctx).Where("repo_url = ?", repoURL).First(&r).Error
	return &r, err
}

func (s *Store) GetRepository(ctx context.Context, id int) (*model.Repository, error) {
	var r model.Repository
	err := s.db.WithContext(ctx).First(&r, id).Error
	return &r, err
}

func (s *Store) ListRepositories(ctx context.Context) ([]model.Repository, error) {
	var repos []model.Repository
	err := s.db.WithContext(ctx).Order("updated_at DESC").Find(&repos).Error
	return repos, err
}

func (s *Store) ListRepositoriesWithLatestReport(ctx context.Context) ([]model.Repository, error) {
	var repos []model.Repository
	err := s.db.WithContext(ctx).
		Where("EXISTS (SELECT 1 FROM vanarana_pipeline_runs WHERE vanarana_pipeline_runs.repo_id = vanarana_repositories.id)").
		Order("updated_at DESC").
		Find(&repos).Error
	return repos, err
}
