package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"vanarana/internal/model"
	"vanarana/internal/store"
)

type PipelineHandler struct {
	store *store.Store
}

func NewPipelineHandler(s *store.Store) *PipelineHandler {
	return &PipelineHandler{store: s}
}

func (h *PipelineHandler) GetPipelineRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline run id")
		return
	}

	pr, err := h.store.GetPipelineRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline run not found")
		return
	}

	repo, err := h.store.GetRepository(r.Context(), pr.RepoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load repository")
		return
	}

	modules, err := h.store.ListModuleReportsByPipelineRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load modules")
		return
	}

	detail := model.PipelineRunDetail{
		PipelineRun:  *pr,
		Repository:   *repo,
		ModuleCount:  len(modules),
	}

	for _, mr := range modules {
		summary, err := h.store.GetModuleReportSummary(r.Context(), mr.ID)
		if err != nil {
			detail.Modules = append(detail.Modules, model.ModuleReportSummary{
				ID: mr.ID, ModuleName: mr.ModuleName, Status: mr.Status,
			})
			continue
		}
		detail.Modules = append(detail.Modules, *summary)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *PipelineHandler) ListByJob(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("repo_url")
	jobName := r.URL.Query().Get("pipeline_job_name")

	if repoURL == "" || jobName == "" {
		writeError(w, http.StatusBadRequest, "repo_url and pipeline_job_name are required")
		return
	}

	repo, err := h.store.GetRepositoryByURL(r.Context(), repoURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	runs, err := h.store.ListRecentPipelineRuns(r.Context(), repo.ID, 365, jobName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipeline runs")
		return
	}

	writeJobRunsResponse(w, r, repo, jobName, runs, h.store)
}

func (h *PipelineHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "repoID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	jobName := r.URL.Query().Get("pipeline_job_name")

	repo, err := h.store.GetRepository(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	runs, err := h.store.ListRecentPipelineRuns(r.Context(), id, days, jobName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipeline runs")
		return
	}

	writeJobRunsResponse(w, r, repo, jobName, runs, h.store)
}

func (h *PipelineHandler) GetRepoTrends(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "repoID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	points, err := h.store.GetRepoModuleTrends(r.Context(), id, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get trends")
		return
	}

	grouped := make(map[string][]model.ModuleTrendPoint)
	for _, p := range points {
		grouped[p.ModuleName] = append(grouped[p.ModuleName], p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"modules": grouped,
		"days":    days,
	})
}

func writeJobRunsResponse(w http.ResponseWriter, r *http.Request, repo *model.Repository, jobName string, runs []model.PipelineRun, st *store.Store) {
	type jobRunItem struct {
		model.PipelineRun
		Modules []model.ModuleReportSummary `json:"modules"`
	}

	var result []jobRunItem
	for _, run := range runs {
		modules, _ := st.ListModuleReportsByPipelineRun(r.Context(), run.ID)
		var summaries []model.ModuleReportSummary
		for _, mr := range modules {
			s, err := st.GetModuleReportSummary(r.Context(), mr.ID)
			if err != nil {
				summaries = append(summaries, model.ModuleReportSummary{
					ID: mr.ID, ModuleName: mr.ModuleName, Status: mr.Status,
				})
				continue
			}
			summaries = append(summaries, *s)
		}
		result = append(result, jobRunItem{PipelineRun: run, Modules: summaries})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repository":        repo,
		"pipeline_job_name": jobName,
		"pipeline_runs":     result,
	})
}
