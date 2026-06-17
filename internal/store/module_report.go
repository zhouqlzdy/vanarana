package store

import (
	"context"
	"encoding/json"
	"time"

	"vanarana/internal/model"

	"gorm.io/gorm/clause"
)

func (s *Store) CreateModuleReport(ctx context.Context, pipelineRunID int, moduleName, junitArchive, jacocoArchive string) (*model.ModuleReport, error) {
	mr := &model.ModuleReport{
		PipelineRunID: pipelineRunID,
		ModuleName:    moduleName,
		JunitArchive:  junitArchive,
		JacocoArchive: jacocoArchive,
		Status:        model.StatusProcessing,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pipeline_run_id"}, {Name: "module_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"junit_archive", "jacoco_archive", "status"}),
	}).Create(mr).Error
	if err != nil {
		return nil, err
	}
	return s.GetModuleReport(ctx, mr.ID)
}

func (s *Store) GetModuleReportByKey(ctx context.Context, pipelineRunID int, moduleName string) (*model.ModuleReport, error) {
	var mr model.ModuleReport
	err := s.db.WithContext(ctx).Where("pipeline_run_id = ? AND module_name = ?", pipelineRunID, moduleName).First(&mr).Error
	return &mr, err
}

func (s *Store) GetModuleReport(ctx context.Context, id int) (*model.ModuleReport, error) {
	var mr model.ModuleReport
	err := s.db.WithContext(ctx).First(&mr, id).Error
	return &mr, err
}

func (s *Store) ListModuleReportsByPipelineRun(ctx context.Context, pipelineRunID int) ([]model.ModuleReport, error) {
	var reports []model.ModuleReport
	err := s.db.WithContext(ctx).Where("pipeline_run_id = ?", pipelineRunID).Order("module_name").Find(&reports).Error
	return reports, err
}

func (s *Store) UpdateModuleReportStatus(ctx context.Context, id int, status string) error {
	return s.db.WithContext(ctx).Model(&model.ModuleReport{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Store) SaveJunitMetrics(ctx context.Context, reportID int, m *model.JunitMetrics) error {
	packagesJSON, _ := json.Marshal(m.Packages)
	m.ReportID = reportID
	m.PackagesJSON = string(packagesJSON)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "report_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_tests", "failures", "ignored", "duration_ms", "success_rate", "packages"}),
	}).Create(m).Error
}

func (s *Store) SaveJacocoMetrics(ctx context.Context, reportID int, m *model.JacocoMetrics) error {
	packagesJSON, _ := json.Marshal(m.Packages)
	m.ReportID = reportID
	m.PackagesJSON = string(packagesJSON)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "report_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"instruction_coverage", "branch_coverage", "line_coverage", "method_coverage", "lines_total", "lines_missed", "packages"}),
	}).Create(m).Error
}

func (s *Store) GetJunitMetrics(ctx context.Context, reportID int) (*model.JunitMetrics, error) {
	var m model.JunitMetrics
	err := s.db.WithContext(ctx).Where("report_id = ?", reportID).First(&m).Error
	if err != nil {
		return nil, err
	}
	if m.PackagesJSON != "" {
		json.Unmarshal([]byte(m.PackagesJSON), &m.Packages)
	}
	return &m, nil
}

func (s *Store) GetJacocoMetrics(ctx context.Context, reportID int) (*model.JacocoMetrics, error) {
	var m model.JacocoMetrics
	err := s.db.WithContext(ctx).Where("report_id = ?", reportID).First(&m).Error
	if err != nil {
		return nil, err
	}
	if m.PackagesJSON != "" {
		json.Unmarshal([]byte(m.PackagesJSON), &m.Packages)
	}
	return &m, nil
}

func (s *Store) GetModuleReportSummary(ctx context.Context, reportID int) (*model.ModuleReportSummary, error) {
	summary := &model.ModuleReportSummary{ID: reportID}
	row := s.db.WithContext(ctx).Raw(`
		SELECT mr.module_name, mr.status,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0), COALESCE(jcm.instruction_coverage, 0)
		 FROM vanarana_module_reports mr
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE mr.id = ?`, reportID).Row()
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
	var points []model.TrendPoint
	err := s.db.WithContext(ctx).Raw(`
		SELECT pr.id AS pipeline_run_id, pr.build_id, pr.triggered_at,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.total_tests - jm.failures, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0), COALESCE(jcm.instruction_coverage, 0)
		 FROM vanarana_module_reports mr
		 JOIN vanarana_pipeline_runs pr ON pr.id = mr.pipeline_run_id
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE pr.repo_id = ? AND mr.module_name = ?
		 ORDER BY pr.triggered_at DESC LIMIT ?`, repoID, moduleName, limit).Scan(&points).Error
	return points, err
}

func (s *Store) GetRepoModuleTrends(ctx context.Context, repoID int, days int) ([]model.ModuleTrendPoint, error) {
	var points []model.ModuleTrendPoint
	err := s.db.WithContext(ctx).Raw(`
		SELECT mr.module_name, pr.build_id, UNIX_TIMESTAMP(pr.triggered_at), pr.triggered_at,
		 COALESCE(jm.total_tests, 0), COALESCE(jm.total_tests - jm.failures, 0), COALESCE(jm.failures, 0),
		 COALESCE(jm.success_rate, 0),
		 COALESCE(jcm.line_coverage, 0), COALESCE(jcm.branch_coverage, 0)
		 FROM vanarana_module_reports mr
		 JOIN vanarana_pipeline_runs pr ON pr.id = mr.pipeline_run_id
		 LEFT JOIN vanarana_junit_metrics jm ON jm.report_id = mr.id
		 LEFT JOIN vanarana_jacoco_metrics jcm ON jcm.report_id = mr.id
		 WHERE pr.repo_id = ? AND pr.triggered_at >= ? AND mr.status = ?
		 ORDER BY mr.module_name, pr.triggered_at ASC`, repoID, time.Now().AddDate(0, 0, -days), model.StatusCompleted).Scan(&points).Error
	return points, err
}
