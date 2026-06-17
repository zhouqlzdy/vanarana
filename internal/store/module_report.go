package store

import (
	"context"
	"encoding/json"
	"fmt"
	"vanarana/internal/model"
)

func (s *Store) CreateModuleReport(ctx context.Context, pipelineRunID int, moduleName, junitArchive, jacocoArchive string) (*model.ModuleReport, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO vanarana_module_reports (pipeline_run_id, module_name, junit_archive, jacoco_archive, status)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE junit_archive = VALUES(junit_archive), jacoco_archive = VALUES(jacoco_archive), status = VALUES(status)`,
		pipelineRunID, moduleName, junitArchive, jacocoArchive, model.StatusProcessing,
	)
	if err != nil {
		return nil, fmt.Errorf("insert module_report: %w", err)
	}

	id, _ := result.LastInsertId()
	if id == 0 {
		return s.GetModuleReportByKey(ctx, pipelineRunID, moduleName)
	}

	return s.GetModuleReport(ctx, int(id))
}

func (s *Store) GetModuleReportByKey(ctx context.Context, pipelineRunID int, moduleName string) (*model.ModuleReport, error) {
	mr := &model.ModuleReport{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pipeline_run_id, module_name, junit_archive, jacoco_archive, status, upload_time
		 FROM vanarana_module_reports WHERE pipeline_run_id = ? AND module_name = ?`,
		pipelineRunID, moduleName,
	).Scan(&mr.ID, &mr.PipelineRunID, &mr.ModuleName, &mr.JunitArchive, &mr.JacocoArchive, &mr.Status, &mr.UploadTime)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

func (s *Store) GetModuleReport(ctx context.Context, id int) (*model.ModuleReport, error) {
	mr := &model.ModuleReport{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, pipeline_run_id, module_name, junit_archive, jacoco_archive, status, upload_time
		 FROM vanarana_module_reports WHERE id = ?`, id,
	).Scan(&mr.ID, &mr.PipelineRunID, &mr.ModuleName, &mr.JunitArchive, &mr.JacocoArchive, &mr.Status, &mr.UploadTime)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

func (s *Store) ListModuleReportsByPipelineRun(ctx context.Context, pipelineRunID int) ([]model.ModuleReport, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, pipeline_run_id, module_name, junit_archive, jacoco_archive, status, upload_time
		 FROM vanarana_module_reports WHERE pipeline_run_id = ? ORDER BY module_name`, pipelineRunID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.ModuleReport
	for rows.Next() {
		var mr model.ModuleReport
		if err := rows.Scan(&mr.ID, &mr.PipelineRunID, &mr.ModuleName, &mr.JunitArchive, &mr.JacocoArchive, &mr.Status, &mr.UploadTime); err != nil {
			return nil, err
		}
		reports = append(reports, mr)
	}
	return reports, rows.Err()
}

func (s *Store) UpdateModuleReportStatus(ctx context.Context, id int, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE vanarana_module_reports SET status = ? WHERE id = ?`, status, id,
	)
	return err
}

func (s *Store) SaveJunitMetrics(ctx context.Context, reportID int, m *model.JunitMetrics) error {
	packagesJSON, _ := json.Marshal(m.Packages)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vanarana_junit_metrics (report_id, total_tests, failures, ignored, duration_ms, success_rate, packages)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE total_tests=VALUES(total_tests), failures=VALUES(failures),
		 ignored=VALUES(ignored), duration_ms=VALUES(duration_ms), success_rate=VALUES(success_rate), packages=VALUES(packages)`,
		reportID, m.TotalTests, m.Failures, m.Ignored, m.DurationMs, m.SuccessRate, string(packagesJSON),
	)
	return err
}

func (s *Store) SaveJacocoMetrics(ctx context.Context, reportID int, m *model.JacocoMetrics) error {
	packagesJSON, _ := json.Marshal(m.Packages)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vanarana_jacoco_metrics (report_id, instruction_coverage, branch_coverage, line_coverage,
		 method_coverage, lines_total, lines_missed, packages)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE instruction_coverage=VALUES(instruction_coverage),
		 branch_coverage=VALUES(branch_coverage), line_coverage=VALUES(line_coverage),
		 method_coverage=VALUES(method_coverage), lines_total=VALUES(lines_total),
		 lines_missed=VALUES(lines_missed), packages=VALUES(packages)`,
		reportID, m.InstructionCoverage, m.BranchCoverage, m.LineCoverage,
		m.MethodCoverage, m.LinesTotal, m.LinesMissed, string(packagesJSON),
	)
	return err
}

func (s *Store) GetJunitMetrics(ctx context.Context, reportID int) (*model.JunitMetrics, error) {
	m := &model.JunitMetrics{ReportID: reportID}
	var packagesJSON []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, total_tests, failures, ignored, duration_ms, success_rate, packages
		 FROM vanarana_junit_metrics WHERE report_id = ?`, reportID,
	).Scan(&m.ID, &m.TotalTests, &m.Failures, &m.Ignored, &m.DurationMs, &m.SuccessRate, &packagesJSON)
	if err != nil {
		return nil, fmt.Errorf("get vanarana_junit_metrics: %w", err)
	}
	if len(packagesJSON) > 0 {
		json.Unmarshal(packagesJSON, &m.Packages)
	}
	return m, nil
}

func (s *Store) GetJacocoMetrics(ctx context.Context, reportID int) (*model.JacocoMetrics, error) {
	m := &model.JacocoMetrics{ReportID: reportID}
	var packagesJSON []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, instruction_coverage, branch_coverage, line_coverage, method_coverage,
		 lines_total, lines_missed, packages
		 FROM vanarana_jacoco_metrics WHERE report_id = ?`, reportID,
	).Scan(&m.ID, &m.InstructionCoverage, &m.BranchCoverage, &m.LineCoverage,
		&m.MethodCoverage, &m.LinesTotal, &m.LinesMissed, &packagesJSON)
	if err != nil {
		return nil, fmt.Errorf("get vanarana_jacoco_metrics: %w", err)
	}
	if len(packagesJSON) > 0 {
		json.Unmarshal(packagesJSON, &m.Packages)
	}
	return m, nil
}

func (s *Store) GetModuleReportSummary(ctx context.Context, reportID int) (*model.ModuleReportSummary, error) {
	summary := &model.ModuleReportSummary{ID: reportID}

	row := s.db.QueryRowContext(ctx,
		`SELECT mr.module_name, mr.status,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0), COALESCE(jcm.instruction_coverage, 0)
		 FROM vanarana_module_reports mr
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE mr.id = ?`, reportID,
	)
	err := row.Scan(
		&summary.ModuleName, &summary.Status,
		&summary.TotalTests, &summary.Failures, &summary.SuccessRate,
		&summary.LineCoverage, &summary.BranchCoverage, &summary.InstructionCoverage,
	)
	if err != nil {
		return nil, err
	}
	summary.Passed = summary.TotalTests - summary.Failures
	return summary, nil
}

func (s *Store) GetTrend(ctx context.Context, repoID int, moduleName string, limit int) ([]model.TrendPoint, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pr.id, pr.build_id, pr.triggered_at,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.total_tests - jm.failures, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0), COALESCE(jcm.instruction_coverage, 0)
		 FROM vanarana_module_reports mr
		 JOIN vanarana_pipeline_runs pr ON pr.id = mr.pipeline_run_id
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE pr.repo_id = ? AND mr.module_name = ?
		 ORDER BY pr.triggered_at DESC LIMIT ?`, repoID, moduleName, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.TrendPoint
	for rows.Next() {
		var p model.TrendPoint
		var triggeredAt interface{}
		if err := rows.Scan(
			&p.PipelineRunID, &p.BuildID, &triggeredAt,
			&p.TotalTests, &p.Passed, &p.Failures,
			&p.SuccessRate, &p.LineCoverage, &p.BranchCoverage, &p.InstructionCoverage,
		); err != nil {
			return nil, err
		}
		if t, ok := triggeredAt.([]byte); ok {
			p.TriggeredAt = string(t)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *Store) GetRepoModuleTrends(ctx context.Context, repoID int, days int) ([]model.ModuleTrendPoint, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT mr.module_name, pr.build_id, UNIX_TIMESTAMP(pr.triggered_at), pr.triggered_at,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.total_tests - jm.failures, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0)
		 FROM vanarana_module_reports mr
		 JOIN vanarana_pipeline_runs pr ON pr.id = mr.pipeline_run_id
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE pr.repo_id = ? AND pr.triggered_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		   AND mr.status = 'completed'
		 ORDER BY mr.module_name, pr.triggered_at ASC`, repoID, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.ModuleTrendPoint
	for rows.Next() {
		var p model.ModuleTrendPoint
		var triggeredAt interface{}
		if err := rows.Scan(
			&p.ModuleName, &p.BuildID, &p.TriggeredAtTs, &triggeredAt,
			&p.TotalTests, &p.Passed, &p.Failures,
			&p.SuccessRate, &p.LineCoverage, &p.BranchCoverage,
		); err != nil {
			return nil, err
		}
		if t, ok := triggeredAt.([]byte); ok {
			p.TriggeredAt = string(t)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
