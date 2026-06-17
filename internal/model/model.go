package model

import "time"

type Repository struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	RepoURL   string    `gorm:"column:repo_url;uniqueIndex;size:512;not null" json:"repo_url"`
	Name      string    `gorm:"size:255;default:''" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Repository) TableName() string { return "vanarana_repositories" }

type PipelineRun struct {
	ID              int          `gorm:"primaryKey;autoIncrement" json:"id"`
	RepoID          int          `gorm:"column:repo_id;uniqueIndex:uk_run;not null" json:"repo_id"`
	PipelineJobName string       `gorm:"column:pipeline_job_name;uniqueIndex:uk_run;size:255;not null" json:"pipeline_job_name"`
	Branch          string       `gorm:"size:255;default:''" json:"branch"`
	CommitHash      string       `gorm:"column:commit_hash;size:64;default:''" json:"commit_hash"`
	BuildID         string       `gorm:"column:build_id;size:128;default:''" json:"build_id"`
	Status          string       `gorm:"size:20;default:processing" json:"status"`
	TriggeredAt     time.Time    `gorm:"autoCreateTime" json:"triggered_at"`
	CreatedAt       time.Time    `gorm:"autoCreateTime" json:"created_at"`
}

func (PipelineRun) TableName() string { return "vanarana_pipeline_runs" }

type ModuleReport struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	PipelineRunID int       `gorm:"column:pipeline_run_id;uniqueIndex:uk_module;not null" json:"pipeline_run_id"`
	ModuleName    string    `gorm:"column:module_name;uniqueIndex:uk_module;size:255;not null" json:"module_name"`
	JunitArchive  string    `gorm:"column:junit_archive;size:512;default:''" json:"junit_archive"`
	JacocoArchive string    `gorm:"column:jacoco_archive;size:512;default:''" json:"jacoco_archive"`
	Status        string    `gorm:"size:20;default:processing" json:"status"`
	UploadTime    time.Time `gorm:"autoCreateTime" json:"upload_time"`
}

func (ModuleReport) TableName() string { return "vanarana_module_reports" }

type JunitMetrics struct {
	ID           int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID     int            `gorm:"column:report_id;uniqueIndex;not null" json:"report_id"`
	TotalTests   int            `gorm:"column:total_tests;default:0" json:"total_tests"`
	Failures     int            `gorm:"default:0" json:"failures"`
	Ignored      int            `gorm:"default:0" json:"ignored"`
	DurationMs   int64          `gorm:"column:duration_ms;default:0" json:"duration_ms"`
	SuccessRate  float64        `gorm:"column:success_rate;default:0" json:"success_rate"`
	PackagesJSON string         `gorm:"column:packages;type:json" json:"-"`
	Packages     []PackageJunit `gorm:"-" json:"packages"`
}

func (JunitMetrics) TableName() string { return "vanarana_junit_metrics" }

type JacocoMetrics struct {
	ID                   int               `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID             int               `gorm:"column:report_id;uniqueIndex;not null" json:"report_id"`
	InstructionCoverage  float64           `gorm:"column:instruction_coverage;default:0" json:"instruction_coverage"`
	BranchCoverage       float64           `gorm:"column:branch_coverage;default:0" json:"branch_coverage"`
	LineCoverage         float64           `gorm:"column:line_coverage;default:0" json:"line_coverage"`
	MethodCoverage       float64           `gorm:"column:method_coverage;default:0" json:"method_coverage"`
	LinesTotal           int               `gorm:"column:lines_total;default:0" json:"lines_total"`
	LinesMissed          int               `gorm:"column:lines_missed;default:0" json:"lines_missed"`
	PackagesJSON         string            `gorm:"column:packages;type:json" json:"-"`
	Packages             []PackageCoverage `gorm:"-" json:"packages"`
}

func (JacocoMetrics) TableName() string { return "vanarana_jacoco_metrics" }

type PackageJunit struct {
	Name       string `json:"name"`
	Tests      int    `json:"tests"`
	Failures   int    `json:"failures"`
	Ignored    int    `json:"ignored"`
	DurationMs int64  `json:"duration_ms"`
}

type PackageCoverage struct {
	Name                string  `json:"name"`
	InstructionCoverage float64 `json:"instruction_coverage"`
	BranchCoverage      float64 `json:"branch_coverage"`
	LineCoverage        float64 `json:"line_coverage"`
	LinesTotal          int     `json:"lines_total"`
	LinesMissed         int     `json:"lines_missed"`
}

type UploadRequest struct {
	RepoURL         string
	ModuleName      string
	PipelineJobName string
	BuildID         string
	Branch          string
	CommitHash      string
}

type PipelineRunDetail struct {
	PipelineRun
	Repository  Repository            `json:"repository"`
	ModuleCount int                   `json:"module_count"`
	Modules     []ModuleReportSummary `json:"modules"`
}

type ModuleReportSummary struct {
	ID                  int     `json:"id"`
	ModuleName          string  `json:"module_name"`
	Status              string  `json:"status"`
	TotalTests          int     `json:"total_tests"`
	Passed              int     `json:"passed"`
	Failures            int     `json:"failures"`
	SuccessRate         float64 `json:"success_rate"`
	LineCoverage        float64 `json:"line_coverage"`
	BranchCoverage      float64 `json:"branch_coverage"`
	InstructionCoverage float64 `json:"instruction_coverage"`
}

type TrendPoint struct {
	PipelineRunID       int     `json:"pipeline_run_id"`
	BuildID             string  `json:"build_id"`
	TriggeredAt         string  `json:"triggered_at"`
	TotalTests          int     `json:"total_tests"`
	Passed              int     `json:"passed"`
	Failures            int     `json:"failures"`
	SuccessRate         float64 `json:"success_rate"`
	LineCoverage        float64 `json:"line_coverage"`
	BranchCoverage      float64 `json:"branch_coverage"`
	InstructionCoverage float64 `json:"instruction_coverage"`
}

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

const (
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)
