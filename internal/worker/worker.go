package worker

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"vanarana/internal/cache"
	"vanarana/internal/model"
	"vanarana/internal/notify"
	"vanarana/internal/parser"
	"vanarana/internal/store"
)

type ParserWorker struct {
	store   *store.Store
	cache   *cache.ReportCache
	jobs    chan int
	neutron *notify.NeutronClient
}

func New(s *store.Store, c *cache.ReportCache, nc *notify.NeutronClient, bufferSize int) *ParserWorker {
	return &ParserWorker{
		store:   s,
		cache:   c,
		jobs:    make(chan int, bufferSize),
		neutron: nc,
	}
}

func (w *ParserWorker) Enqueue(moduleReportID int) {
	select {
	case w.jobs <- moduleReportID:
	default:
		log.Printf("worker queue full, dropping report %d", moduleReportID)
	}
}

func (w *ParserWorker) Run(ctx context.Context, concurrency int) {
	for i := 0; i < concurrency; i++ {
		go w.workLoop(ctx)
	}
}

func (w *ParserWorker) workLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reportID := <-w.jobs:
			w.process(ctx, reportID)
		}
	}
}

func (w *ParserWorker) process(ctx context.Context, reportID int) {
	report, err := w.store.GetModuleReport(ctx, reportID)
	if err != nil {
		log.Printf("worker: get report %d: %v", reportID, err)
		return
	}

	extractAndParse := func(archiveName, reportType string) (*model.JunitMetrics, *model.JacocoMetrics, error) {
		dir, err := w.cache.GetOrExtract(report.PipelineRunID, report.ModuleName, reportType)
		if err != nil {
			return nil, nil, err
		}

		var junitMetrics *model.JunitMetrics
		var jacocoMetrics *model.JacocoMetrics

		if reportType == "junit" {
			testDir := findTestDir(dir)
			if testDir == "" {
				testDir = dir
			}
			indexPath := filepath.Join(testDir, "index.html")
			f, err := os.Open(indexPath)
			if err != nil {
				return nil, nil, err
			}
			defer f.Close()
			junitMetrics, err = parser.ParseJunit(f)
			if err != nil {
				return nil, nil, err
			}
		}

		if reportType == "jacoco" {
			indexDir := findJaCoCoDir(dir)
			if indexDir == "" {
				indexDir = dir
			}
			indexPath := filepath.Join(indexDir, "index.html")
			f, err := os.Open(indexPath)
			if err != nil {
				return nil, nil, err
			}
			defer f.Close()
			jacocoMetrics, err = parser.ParseJacoco(f)
			if err != nil {
				return nil, nil, err
			}
		}

		return junitMetrics, jacocoMetrics, nil
	}

	// Parse Junit
	junitMetrics, _, err := extractAndParse(report.JunitArchive, "junit")
	if err != nil {
		log.Printf("worker: parse junit report %d: %v", reportID, err)
		w.store.UpdateModuleReportStatus(ctx, reportID, model.StatusFailed)
		return
	}
	if junitMetrics != nil {
		junitMetrics.ReportID = reportID
		if err := w.store.SaveJunitMetrics(ctx, reportID, junitMetrics); err != nil {
			log.Printf("worker: save junit metrics %d: %v", reportID, err)
		}
	}

	// Parse Jacoco
	_, jacocoMetrics, err := extractAndParse(report.JacocoArchive, "jacoco")
	if err != nil {
		log.Printf("worker: parse jacoco report %d: %v", reportID, err)
		w.store.UpdateModuleReportStatus(ctx, reportID, model.StatusFailed)
		return
	}
	if jacocoMetrics != nil {
		jacocoMetrics.ReportID = reportID
		if err := w.store.SaveJacocoMetrics(ctx, reportID, jacocoMetrics); err != nil {
			log.Printf("worker: save jacoco metrics %d: %v", reportID, err)
		}
	}

	w.store.UpdateModuleReportStatus(ctx, reportID, model.StatusCompleted)

	// Update pipeline run status if all modules completed
	w.updatePipelineRunStatus(ctx, report.PipelineRunID)
}

func findTestDir(dir string) string {
	candidates := []string{
		filepath.Join(dir, "reports", "tests", "test"),
		filepath.Join(dir, "tests", "test"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
			return c
		}
	}
	return ""
}

func findJaCoCoDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(dir, e.Name())
		if _, err := os.Stat(filepath.Join(subDir, "index.html")); err == nil {
			return subDir
		}
	}
	return ""
}

func (w *ParserWorker) updatePipelineRunStatus(ctx context.Context, pipelineRunID int) {
	modules, err := w.store.ListModuleReportsByPipelineRun(ctx, pipelineRunID)
	if err != nil {
		return
	}
	for _, m := range modules {
		if m.Status != model.StatusCompleted && m.Status != model.StatusFailed {
			return
		}
	}
	w.store.UpdatePipelineRunStatus(ctx, pipelineRunID, model.StatusCompleted)

	pr, err := w.store.GetPipelineRun(ctx, pipelineRunID)
	if err == nil && pr != nil {
		w.neutron.SendReportLink(pr.PipelineJobName)
	}
}
