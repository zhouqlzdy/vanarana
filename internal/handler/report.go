package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"vanarana/internal/cache"
	"vanarana/internal/store"
)

type ReportHandler struct {
	store *store.Store
	cache *cache.ReportCache
}

func NewReportHandler(s *store.Store, c *cache.ReportCache) *ReportHandler {
	return &ReportHandler{store: s, cache: c}
}

func (h *ReportHandler) GetModuleReport(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	mr, err := h.store.GetModuleReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	junit, _ := h.store.GetJunitMetrics(r.Context(), id)
	jacoco, _ := h.store.GetJacocoMetrics(r.Context(), id)

	resp := map[string]interface{}{
		"module_report": mr,
		"junit":         junit,
		"jacoco":        jacoco,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ReportHandler) ServeReportHTML(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid report id", http.StatusBadRequest)
		return
	}

	reportType := chi.URLParam(r, "type")
	if reportType != "junit" && reportType != "jacoco" {
		http.Error(w, "type must be junit or jacoco", http.StatusBadRequest)
		return
	}

	mr, err := h.store.GetModuleReport(r.Context(), id)
	if err != nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}

	cacheDir, err := h.cache.GetOrExtract(mr.PipelineRunID, mr.ModuleName, reportType)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to extract report: %v", err), http.StatusInternalServerError)
		return
	}

	if reportType == "junit" {
		testDir := findJunitTestDir(cacheDir)
		if testDir != "" {
			cacheDir = testDir
		}
	}
	if reportType == "jacoco" {
		subDir := findWrapperDir(cacheDir)
		if subDir != "" {
			cacheDir = subDir
		}
	}

	prefix := fmt.Sprintf("/api/v1/module-reports/%d/%s/html", id, reportType)
	fs := http.StripPrefix(prefix, http.FileServer(http.Dir(cacheDir)))
	fs.ServeHTTP(w, r)
}

func findJunitTestDir(dir string) string {
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

func findWrapperDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		if _, err := os.Stat(filepath.Join(sub, "index.html")); err == nil {
			return sub
		}
	}
	return ""
}

func (h *ReportHandler) GetTrend(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("repo_url")
	moduleName := r.URL.Query().Get("module_name")
	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	if repoURL == "" || moduleName == "" {
		writeError(w, http.StatusBadRequest, "repo_url and module_name are required")
		return
	}

	repo, err := h.store.GetRepositoryByURL(r.Context(), repoURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	points, err := h.store.GetTrend(r.Context(), repo.ID, moduleName, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get trend")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module_name": moduleName,
		"points":      points,
	})
}
