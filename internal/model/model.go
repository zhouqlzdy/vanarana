package model

import "time"

// Repository represents a code repository.
type Repository struct {
	ID        int       `json:"id"`
	RepoURL   string    `json:"repo_url"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PipelineRun groups module reports from a single CI pipeline execution.
type PipelineRun struct {
	ID               int       `json:"id"`
	RepoID           int       `json:"repo_id"`
	PipelineJobName  string    `json:"pipeline_job_name"`
	Branch           string    `json:"branch"`
	CommitHash       string    `json:"commit_hash"`
	BuildID          string    `json:"build_id"`
	Status           string    `json:"status"`
	TriggeredAt      time.Time `json:"triggered_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// ReportStatus values.
const (
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// ModuleReport is a single module's uploaded report within a pipeline run.
type ModuleReport struct {
	ID             int       `json:"id"`
	PipelineRunID  int       `json:"pipeline_run_id"`
	ModuleName     string    `json:"module_name"`
	JunitArchive   string    `json:"junit_archive"`  // path within archive dir
	JacocoArchive  string    `json:"jacoco_archive"` // path within archive dir
	Status         string    `json:"status"`
	UploadTime     time.Time `json:"upload_time"`
}

// JunitMetrics holds parsed JUnit HTML summary.
type JunitMetrics struct {
	ID          int              `json:"id,omitempty"`
	ReportID    int              `json:"report_id"`
	TotalTests  int              `json:"total_tests"`
	Failures    int              `json:"failures"`
	Ignored     int              `json:"ignored"`
	DurationMs  int64            `json:"duration_ms"`
	SuccessRate float64          `json:"success_rate"`
	Packages    []PackageJunit   `json:"packages"`
}

// PackageJunit is a single package's JUnit result.
type PackageJunit struct {
	Name       string  `json:"name"`
	Tests      int     `json:"tests"`
	Failures   int     `json:"failures"`
	Ignored    int     `json:"ignored"`
	DurationMs int64   `json:"duration_ms"`
}

// JacocoMetrics holds parsed JaCoCo HTML coverage summary.
type JacocoMetrics struct {
	ID                   int                 `json:"id,omitempty"`
	ReportID             int                 `json:"report_id"`
	InstructionCoverage  float64             `json:"instruction_coverage"`
	BranchCoverage       float64             `json:"branch_coverage"`
	LineCoverage         float64             `json:"line_coverage"`
	MethodCoverage       float64             `json:"method_coverage"`
	LinesTotal           int                 `json:"lines_total"`
	LinesMissed          int                 `json:"lines_missed"`
	Packages             []PackageCoverage   `json:"packages"`
}

// PackageCoverage is a single package's coverage data.
type PackageCoverage struct {
	Name                string  `json:"name"`
	InstructionCoverage float64 `json:"instruction_coverage"`
	BranchCoverage      float64 `json:"branch_coverage"`
	LineCoverage        float64 `json:"line_coverage"`
	LinesTotal          int     `json:"lines_total"`
	LinesMissed         int     `json:"lines_missed"`
}

// UploadRequest is what the CI pipeline sends.
type UploadRequest struct {
	RepoURL         string
	ModuleName      string
	PipelineJobName string
	BuildID         string
	Branch          string
	CommitHash      string
}

// PipelineRunDetail is the aggregated view for the pipeline query page.
type PipelineRunDetail struct {
	PipelineRun
	Repository  Repository               `json:"repository"`
	ModuleCount int                      `json:"module_count"`
	Modules     []ModuleReportSummary    `json:"modules"`
}

// ModuleReportSummary is a lightweight summary for listing.
type ModuleReportSummary struct {
	ID                   int     `json:"id"`
	ModuleName           string  `json:"module_name"`
	Status               string  `json:"status"`
	TotalTests           int     `json:"total_tests"`
	Passed               int     `json:"passed"`
	Failures             int     `json:"failures"`
	SuccessRate          float64 `json:"success_rate"`
	LineCoverage         float64 `json:"line_coverage"`
	BranchCoverage       float64 `json:"branch_coverage"`
	InstructionCoverage  float64 `json:"instruction_coverage"`
}

// TrendPoint is a single data point in a trend series.
type TrendPoint struct {
	PipelineRunID        int     `json:"pipeline_run_id"`
	BuildID              string  `json:"build_id"`
	TriggeredAt          string  `json:"triggered_at"`
	TotalTests           int     `json:"total_tests"`
	Passed               int     `json:"passed"`
	Failures             int     `json:"failures"`
	SuccessRate          float64 `json:"success_rate"`
	LineCoverage         float64 `json:"line_coverage"`
	BranchCoverage       float64 `json:"branch_coverage"`
	InstructionCoverage  float64 `json:"instruction_coverage"`
}

// ModuleTrendPoint is a lightweight point for chart rendering.
type ModuleTrendPoint struct {
	ModuleName     string  `json:"module_name"`
	BuildID        string  `json:"build_id"`
	TriggeredAt    string  `json:"triggered_at"`
	TriggeredAtTs  int64   `json:"triggered_at_ts"`
	TotalTests     int     `json:"total_tests"`
	Passed         int     `json:"passed"`
	Failures       int     `json:"failures"`
	SuccessRate    float64 `json:"success_rate"`
	LineCoverage   float64 `json:"line_coverage"`
	BranchCoverage float64 `json:"branch_coverage"`
}
