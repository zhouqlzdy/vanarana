package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"vanarana/internal/archive"
	"vanarana/internal/cache"
	"vanarana/internal/config"
	"vanarana/internal/handler"
	"vanarana/internal/notify"
	"vanarana/internal/store"
	"vanarana/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	st, err := store.New(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer st.Close()

	if err := st.RunMigrations("migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	archiveStore, err := archive.New(cfg.ArchiveDir())
	if err != nil {
		log.Fatalf("create archive store: %v", err)
	}

	reportCache, err := cache.New(cfg.CacheDir(), archiveStore, cfg.CacheMaxMB)
	if err != nil {
		log.Fatalf("create report cache: %v", err)
	}

	neutronClient := notify.New(cfg.NeutronAPIURL, cfg.VanaranaBaseURL)
	parserWorker := worker.New(st, reportCache, neutronClient, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parserWorker.Run(ctx, 4)

	pagesHandler, err := handler.NewPagesHandler(st)
	if err != nil {
		log.Fatalf("create pages handler: %v", err)
	}

	uploadHandler := handler.NewUploadHandler(st, archiveStore, parserWorker)
	pipelineHandler := handler.NewPipelineHandler(st)
	reportHandler := handler.NewReportHandler(st, reportCache)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/static/*", pagesHandler.StaticFS().ServeHTTP)

	r.Get("/", pagesHandler.Home)
	r.Get("/pipeline", pagesHandler.PipelinePage)
	r.Get("/report/{id}", pagesHandler.ReportPage)
	r.Get("/repo/{id}", pagesHandler.RepoPage)
	r.Get("/run/{jobName}/{buildId}", pagesHandler.RunPage)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/reports", uploadHandler.Handle)

		r.Get("/pipeline-runs/{id}", pipelineHandler.GetPipelineRun)
		r.Get("/pipeline-runs", pipelineHandler.ListByJob)

		r.Get("/repositories/{repoID}/pipeline-runs", pipelineHandler.ListRecent)
		r.Get("/repositories/{repoID}/trends", pipelineHandler.GetRepoTrends)

		r.Get("/module-reports/{id}", reportHandler.GetModuleReport)
		r.Get("/module-reports/{id}/{type}/html/*", reportHandler.ServeReportHTML)
		r.Get("/trends", reportHandler.GetTrend)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("vanarana server starting on :%d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
