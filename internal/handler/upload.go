package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"vanarana/internal/archive"
	"vanarana/internal/model"
	"vanarana/internal/store"
	"vanarana/internal/worker"
)

type UploadHandler struct {
	store  *store.Store
	arch   *archive.Store
	worker *worker.ParserWorker
}

func NewUploadHandler(s *store.Store, a *archive.Store, w *worker.ParserWorker) *UploadHandler {
	return &UploadHandler{store: s, arch: a, worker: w}
}

type uploadResponse struct {
	PipelineRunID int    `json:"pipeline_run_id"`
	ModuleReportID int   `json:"module_report_id"`
	RepoID         int   `json:"repo_id"`
	RepoURL        string `json:"repo_url"`
	RepoName       string `json:"repo_name"`
	JobName        string `json:"pipeline_job_name"`
	BuildID        string `json:"build_id"`
	ModuleName     string `json:"module_name"`
	Status         string `json:"status"`
}

func (h *UploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request too large")
		return
	}

	req := model.UploadRequest{
		RepoURL:         r.FormValue("repo_url"),
		ModuleName:      r.FormValue("module_name"),
		PipelineJobName: r.FormValue("pipeline_job_name"),
		BuildID:         r.FormValue("build_id"),
		Branch:          r.FormValue("branch"),
		CommitHash:      r.FormValue("commit_hash"),
	}

	if req.RepoURL == "" || req.ModuleName == "" || req.PipelineJobName == "" {
		writeError(w, http.StatusBadRequest, "repo_url, module_name, pipeline_job_name are required")
		return
	}

	// Validate file fields exist
	if _, _, err := r.FormFile("junit"); err != nil {
		writeError(w, http.StatusBadRequest, "junit file is required")
		return
	}
	if _, _, err := r.FormFile("jacoco"); err != nil {
		writeError(w, http.StatusBadRequest, "jacoco file is required")
		return
	}

	// Upsert repository
	repo, err := h.store.UpsertRepository(r.Context(), req.RepoURL)
	if err != nil {
		log.Printf("upsert repo: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create repository")
		return
	}

	// Upsert pipeline run
	pipelineRun, err := h.store.UpsertPipelineRun(
		r.Context(), repo.ID, req.PipelineJobName, req.BuildID, req.Branch, req.CommitHash,
	)
	if err != nil {
		log.Printf("upsert pipeline_run: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create pipeline run")
		return
	}

	// Save archive files
	junitName, err := saveFile(r, "junit", h.arch, pipelineRun.ID, req.ModuleName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save junit: %v", err))
		return
	}
	jacocoName, err := saveFile(r, "jacoco", h.arch, pipelineRun.ID, req.ModuleName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("save jacoco: %v", err))
		return
	}

	// Create module report
	moduleReport, err := h.store.CreateModuleReport(
		r.Context(), pipelineRun.ID, req.ModuleName, junitName, jacocoName,
	)
	if err != nil {
		log.Printf("create module_report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create module report")
		return
	}

	// Enqueue async parsing
	h.worker.Enqueue(moduleReport.ID)

	resp := uploadResponse{
		PipelineRunID: pipelineRun.ID,
		ModuleReportID: moduleReport.ID,
		RepoID:        repo.ID,
		RepoURL:       repo.RepoURL,
		RepoName:      repo.Name,
		JobName:       req.PipelineJobName,
		BuildID:       req.BuildID,
		ModuleName:    req.ModuleName,
		Status:        model.StatusProcessing,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func saveFile(r *http.Request, field string, arch *archive.Store, pipelineRunID int, moduleName string) (string, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Validate gzip magic bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n < 2 || buf[0] != 0x1f || buf[1] != 0x8b {
		return "", fmt.Errorf("%s is not a valid gzip file", field)
	}
	// Seek back to start
	file.Seek(0, io.SeekStart)

	return arch.Save(pipelineRunID, moduleName, field, file)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
