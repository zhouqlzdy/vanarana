package store

import (
	"context"
	"time"

	"vanarana/internal/model"

	"gorm.io/gorm/clause"
)

func (s *Store) UpsertPipelineRun(ctx context.Context, repoID int, jobName, buildID, branch, commitHash string) (*model.PipelineRun, error) {
	pr := &model.PipelineRun{
		RepoID:          repoID,
		PipelineJobName: jobName,
		BuildID:         buildID,
		Branch:          branch,
		CommitHash:      commitHash,
		Status:          model.StatusProcessing,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "repo_id"}, {Name: "pipeline_job_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"build_id", "branch", "commit_hash"}),
	}).Create(pr).Error
	if err != nil {
		return nil, err
	}
	return s.GetPipelineRun(ctx, pr.ID)
}

func (s *Store) GetPipelineRunByKey(ctx context.Context, repoID int, jobName string) (*model.PipelineRun, error) {
	var pr model.PipelineRun
	err := s.db.WithContext(ctx).Where("repo_id = ? AND pipeline_job_name = ?", repoID, jobName).First(&pr).Error
	return &pr, err
}

func (s *Store) GetPipelineRun(ctx context.Context, id int) (*model.PipelineRun, error) {
	var pr model.PipelineRun
	err := s.db.WithContext(ctx).First(&pr, id).Error
	return &pr, err
}

func (s *Store) GetPipelineRunByJobName(ctx context.Context, jobName string) (*model.PipelineRun, error) {
	var pr model.PipelineRun
	err := s.db.WithContext(ctx).Where("pipeline_job_name = ?", jobName).First(&pr).Error
	return &pr, err
}

func (s *Store) ListPipelineRunsByJob(
	ctx context.Context, repoID int, jobName string,
) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	err := s.db.WithContext(ctx).
		Where("repo_id = ? AND pipeline_job_name = ?", repoID, jobName).
		Order("triggered_at DESC").Limit(50).Find(&runs).Error
	return runs, err
}

func (s *Store) UpdatePipelineRunStatus(ctx context.Context, id int, status string) error {
	return s.db.WithContext(ctx).Model(&model.PipelineRun{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Store) ListRecentPipelineRuns(
	ctx context.Context, repoID int, days int, jobName string,
) ([]model.PipelineRun, error) {
	db := s.db.WithContext(ctx).Where("repo_id = ? AND triggered_at >= ?",
		repoID, time.Now().AddDate(0, 0, -days))
	if jobName != "" {
		db = db.Where("pipeline_job_name = ?", jobName)
	}
	var runs []model.PipelineRun
	err := db.Order("triggered_at DESC").Limit(50).Find(&runs).Error
	return runs, err
}

func (s *Store) ListPipelineRunsByJobName(
	ctx context.Context, jobName string, days int,
) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	err := s.db.WithContext(ctx).
		Where("pipeline_job_name = ? AND triggered_at >= ?", jobName, time.Now().AddDate(0, 0, -days)).
		Order("triggered_at DESC").Limit(50).Find(&runs).Error
	return runs, err
}
