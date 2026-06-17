package handler

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"vanarana/internal/model"
	"vanarana/internal/store"
	"vanarana/web"
)

type PagesHandler struct {
	store         *store.Store
	homeTmpl      *template.Template
	pipelineTmpl  *template.Template
	reportTmpl    *template.Template
	repoTmpl      *template.Template
	runTmpl       *template.Template
}

func NewPagesHandler(s *store.Store) (*PagesHandler, error) {
	parsePattern := func(files ...string) *template.Template {
		tmpl, err := template.New("base").ParseFS(web.TemplatesFS, files...)
		if err != nil {
			panic(fmt.Errorf("parse templates %v: %w", files, err))
		}
		return tmpl
	}

	return &PagesHandler{
		store:        s,
		homeTmpl:     parsePattern("templates/base.html", "templates/home.html"),
		pipelineTmpl: parsePattern("templates/base.html", "templates/pipeline.html"),
		reportTmpl:   parsePattern("templates/base.html", "templates/report.html"),
		repoTmpl:     parsePattern("templates/base.html", "templates/repo.html"),
		runTmpl:      parsePattern("templates/base.html", "templates/run.html"),
	}, nil
}

func (h *PagesHandler) StaticFS() http.Handler {
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
}

func (h *PagesHandler) Home(w http.ResponseWriter, r *http.Request) {
	repos, _ := h.store.ListRepositoriesWithLatestReport(r.Context())
	h.homeTmpl.ExecuteTemplate(w, "base", map[string]interface{}{
		"Repos": repos,
	})
}

func (h *PagesHandler) PipelinePage(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("repo_url")
	jobName := r.URL.Query().Get("pipeline_job_name")
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	repos, _ := h.store.ListRepositories(r.Context())

	data := map[string]interface{}{
		"Repos":        repos,
		"SelectedRepo": repoURL,
		"SelectedJob":  jobName,
		"SelectedDays": days,
	}

	if repoURL != "" {
		data["Searched"] = true
		repo, err := h.store.GetRepositoryByURL(r.Context(), repoURL)
		if err == nil {
			data["RepoName"] = repo.Name
			runs, _ := h.store.ListRecentPipelineRuns(r.Context(), repo.ID, days, jobName)
			data["Runs"] = h.enrichRuns(r.Context(), runs)
		}
	}

	h.pipelineTmpl.ExecuteTemplate(w, "base", data)
}

func (h *PagesHandler) enrichRuns(ctx context.Context, runs []model.PipelineRun) []runWithModules {
	var enriched []runWithModules
	for _, run := range runs {
		modules, _ := h.store.ListModuleReportsByPipelineRun(ctx, run.ID)
		var summaries []moduleSummary
		for _, mr := range modules {
			s, err := h.store.GetModuleReportSummary(ctx, mr.ID)
			if err != nil {
				continue
			}
			summaries = append(summaries, moduleSummary{
				ID: s.ID, ModuleName: s.ModuleName, Status: s.Status,
				TotalTests: s.TotalTests, Passed: s.Passed, Failures: s.Failures,
				SuccessRate: s.SuccessRate, LineCoverage: s.LineCoverage,
				BranchCoverage: s.BranchCoverage, InstructionCoverage: s.InstructionCoverage,
			})
		}
		enriched = append(enriched, runWithModules{
			ID: run.ID, BuildID: run.BuildID, Branch: run.Branch,
			CommitHash: run.CommitHash, Status: run.Status, TriggeredAt: run.TriggeredAt,
			PipelineJobName: run.PipelineJobName,
			Modules: summaries,
		})
	}
	return enriched
}

type runWithModules struct {
	ID               int
	BuildID          string
	Branch           string
	CommitHash       string
	Status           string
	TriggeredAt      time.Time
	PipelineJobName  string
	Modules          []moduleSummary
}

type moduleSummary struct {
	ID                  int
	ModuleName          string
	Status              string
	TotalTests          int
	Passed              int
	Failures            int
	SuccessRate         float64
	LineCoverage        float64
	BranchCoverage      float64
	InstructionCoverage float64
}

func (h *PagesHandler) ReportPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	reportID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "invalid report id", http.StatusBadRequest)
		return
	}

	mr, err := h.store.GetModuleReport(r.Context(), reportID)
	if err != nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}

	pr, _ := h.store.GetPipelineRun(r.Context(), mr.PipelineRunID)
	repo, _ := h.store.GetRepository(r.Context(), pr.RepoID)

	junit, _ := h.store.GetJunitMetrics(r.Context(), reportID)
	jacoco, _ := h.store.GetJacocoMetrics(r.Context(), reportID)

	passed := 0
	if junit != nil {
		passed = junit.TotalTests - junit.Failures - junit.Ignored
	}

	durationDisplay := "N/A"
	if junit != nil && junit.DurationMs > 0 {
		durationDisplay = formatDuration(junit.DurationMs)
	}

	h.reportTmpl.ExecuteTemplate(w, "base", map[string]interface{}{
		"ModuleReport":    mr,
		"PipelineRun":     pr,
		"Repository":      repo,
		"Junit":           junit,
		"Jacoco":          jacoco,
		"Passed":          passed,
		"DurationDisplay": durationDisplay,
	})
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	if ms < 60000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	mins := ms / 60000
	secs := float64(ms%60000) / 1000
	return fmt.Sprintf("%dm %.0fs", mins, secs)
}

func (h *PagesHandler) RepoPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid repo id", http.StatusBadRequest)
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	repo, err := h.store.GetRepository(r.Context(), id)
	if err != nil {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	runs, _ := h.store.ListRecentPipelineRuns(r.Context(), id, days, "")

	h.repoTmpl.ExecuteTemplate(w, "base", map[string]interface{}{
		"Repository":    repo,
		"Days":          days,
		"Runs":          h.enrichRuns(r.Context(), runs),
	})
}

func (h *PagesHandler) RunPage(w http.ResponseWriter, r *http.Request) {
	jobName := chi.URLParam(r, "jobName")

	pr, err := h.store.GetPipelineRunByJobName(r.Context(), jobName)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	repo, _ := h.store.GetRepository(r.Context(), pr.RepoID)

	runs := []model.PipelineRun{*pr}

	h.runTmpl.ExecuteTemplate(w, "base", map[string]interface{}{
		"JobName":       jobName,
		"Repository":    repo,
		"Runs":          h.enrichRuns(r.Context(), runs),
	})
}
