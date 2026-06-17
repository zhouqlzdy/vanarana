package store

import (
	"context"

	"vanarana/internal/model"
)

func (s *Store) UpsertRepository(ctx context.Context, repoURL string) (*model.Repository, error) {
	// Extract name from URL (last path segment)
	name := repoURL
	for i := len(repoURL) - 1; i >= 0; i-- {
		if repoURL[i] == '/' {
			name = repoURL[i+1:]
			break
		}
	}
	// Strip .git suffix
	if len(name) > 4 && name[len(name)-4:] == ".git" {
		name = name[:len(name)-4]
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vanarana_repositories (repo_url, name) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		repoURL, name,
	)
	if err != nil {
		return nil, err
	}

	return s.GetRepositoryByURL(ctx, repoURL)
}

func (s *Store) GetRepositoryByURL(ctx context.Context, repoURL string) (*model.Repository, error) {
	r := &model.Repository{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_url, name, created_at, updated_at FROM vanarana_repositories WHERE repo_url = ?`,
		repoURL,
	).Scan(&r.ID, &r.RepoURL, &r.Name, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) GetRepository(ctx context.Context, id int) (*model.Repository, error) {
	r := &model.Repository{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, repo_url, name, created_at, updated_at FROM vanarana_repositories WHERE id = ?`,
		id,
	).Scan(&r.ID, &r.RepoURL, &r.Name, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) ListRepositories(ctx context.Context) ([]model.Repository, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo_url, name, created_at, updated_at FROM vanarana_repositories ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []model.Repository
	for rows.Next() {
		var r model.Repository
		if err := rows.Scan(&r.ID, &r.RepoURL, &r.Name, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *Store) ListRepositoriesWithLatestReport(ctx context.Context) ([]model.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.repo_url, r.name, r.created_at, r.updated_at
		FROM vanarana_repositories r
		WHERE EXISTS (
			SELECT 1 FROM vanarana_pipeline_runs pr WHERE pr.repo_id = r.id
		)
		ORDER BY r.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []model.Repository
	for rows.Next() {
		var r model.Repository
		if err := rows.Scan(&r.ID, &r.RepoURL, &r.Name, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
