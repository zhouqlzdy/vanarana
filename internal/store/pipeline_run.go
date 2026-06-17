package store

import (
	"context"
	"fmt"
	"vanarana/internal/model"
)

func (s *Store) UpsertPipelineRun(ctx context.Context, repoID int, jobName, buildID, branch, commitHash string) (*model.PipelineRun, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO vanarana_pipeline_runs (repo_id, pipeline_job_name, build_id, branch, commit_hash, status)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE build_id = VALUES(build_id), branch = VALUES(branch), commit_hash = VALUES(commit_hash)`,
		repoID, jobName, buildID, branch, commitHash, model.StatusProcessing,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert pipeline_run: %w", err)
	}

	id, _ := result.LastInsertId()
	if id == 0 {
		return s.GetPipelineRunByKey(ctx, repoID, jobName)
	}

	return s.GetPipelineRun(ctx, int(id))
}

func (s *Store) GetPipelineRunByKey(ctx context.Context, repoID int, jobName string) (*model.PipelineRun, error) {
	pr := &model.PipelineRun{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE repo_id = ? AND pipeline_job_name = ?`,
		repoID, jobName,
	).Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *Store) GetPipelineRun(ctx context.Context, id int) (*model.PipelineRun, error) {
	pr := &model.PipelineRun{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE id = ?`, id,
	).Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *Store) GetPipelineRunByJobName(ctx context.Context, jobName string) (*model.PipelineRun, error) {
	pr := &model.PipelineRun{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE pipeline_job_name = ?`, jobName,
	).Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *Store) ListPipelineRunsByJob(
	ctx context.Context, repoID int, jobName string,
) ([]model.PipelineRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE repo_id = ? AND pipeline_job_name = ?
		 ORDER BY triggered_at DESC LIMIT 50`,
		repoID, jobName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []model.PipelineRun
	for rows.Next() {
		var pr model.PipelineRun
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, pr)
	}
	return runs, rows.Err()
}

func (s *Store) UpdatePipelineRunStatus(ctx context.Context, id int, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE vanarana_pipeline_runs SET status = ? WHERE id = ?`, status, id,
	)
	return err
}

func (s *Store) ListRecentPipelineRuns(
	ctx context.Context, repoID int, days int, jobName string,
) ([]model.PipelineRun, error) {
	query := `SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE repo_id = ? AND triggered_at >= DATE_SUB(NOW(), INTERVAL ? DAY)`
	args := []interface{}{repoID, days}

	if jobName != "" {
		query += ` AND pipeline_job_name = ?`
		args = append(args, jobName)
	}

	query += ` ORDER BY triggered_at DESC LIMIT 50`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []model.PipelineRun
	for rows.Next() {
		var pr model.PipelineRun
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, pr)
	}
	return runs, rows.Err()
}

func (s *Store) ListPipelineRunsByJobName(
	ctx context.Context, jobName string, days int,
) ([]model.PipelineRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo_id, pipeline_job_name, branch, commit_hash, build_id, status, triggered_at, created_at
		 FROM vanarana_pipeline_runs WHERE pipeline_job_name = ? AND triggered_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		 ORDER BY triggered_at DESC LIMIT 50`,
		jobName, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []model.PipelineRun
	for rows.Next() {
		var pr model.PipelineRun
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.PipelineJobName, &pr.Branch, &pr.CommitHash, &pr.BuildID, &pr.Status, &pr.TriggeredAt, &pr.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, pr)
	}
	return runs, rows.Err()
}
