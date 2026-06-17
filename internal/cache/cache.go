package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"vanarana/internal/archive"
)

type entry struct {
	path     string
	size     int64
	lastUsed time.Time
}

type ReportCache struct {
	cacheDir   string
	archive    *archive.Store
	maxSize    int64

	mu      sync.Mutex
	entries map[string]*entry
}

func New(cacheDir string, archiveStore *archive.Store, maxSizeMB int64) (*ReportCache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &ReportCache{
		cacheDir: cacheDir,
		archive:  archiveStore,
		maxSize:  maxSizeMB * 1024 * 1024,
		entries:  make(map[string]*entry),
	}, nil
}

func (c *ReportCache) GetOrExtract(pipelineRunID int, moduleName, reportType string) (string, error) {
	key := fmt.Sprintf("%d_%s_%s", pipelineRunID, moduleName, reportType)

	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		e.lastUsed = time.Now()
		c.mu.Unlock()
		return e.path, nil
	}
	c.mu.Unlock()

	return c.extract(key, pipelineRunID, moduleName, reportType)
}

func (c *ReportCache) extract(key string, pipelineRunID int, moduleName, reportType string) (string, error) {
	archiveName := fmt.Sprintf("%d_%s_%s.tar.gz", pipelineRunID, moduleName, reportType)
	destDir := filepath.Join(c.cacheDir, key)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Check if already extracted but not in memory (e.g., after restart)
	if _, err := os.Stat(filepath.Join(destDir, "index.html")); err == nil {
		size := dirSize(destDir)
		c.mu.Lock()
		c.entries[key] = &entry{path: destDir, size: size, lastUsed: time.Now()}
		c.evict()
		c.mu.Unlock()
		return destDir, nil
	}

	src, err := c.archive.Open(archiveName)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer src.Close()

	if err := extractTarGz(src, destDir); err != nil {
		os.RemoveAll(destDir)
		return "", fmt.Errorf("extract tar.gz: %w", err)
	}

	size := dirSize(destDir)

	c.mu.Lock()
	c.entries[key] = &entry{path: destDir, size: size, lastUsed: time.Now()}
	c.evict()
	c.mu.Unlock()

	return destDir, nil
}

func (c *ReportCache) evict() {
	var total int64
	for _, e := range c.entries {
		total += e.size
	}

	if total <= c.maxSize {
		return
	}

	type keyTime struct {
		key      string
		lastUsed time.Time
	}
	var sorted []keyTime
	for k, e := range c.entries {
		sorted = append(sorted, keyTime{k, e.lastUsed})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].lastUsed.Before(sorted[j].lastUsed)
	})

	for _, kt := range sorted {
		if total <= c.maxSize {
			break
		}
		e := c.entries[kt.key]
		os.RemoveAll(e.path)
		total -= e.size
		delete(c.entries, kt.key)
	}
}

func extractTarGz(r io.Reader, dest string) error {
	return archive.ExtractTarGz(r, dest)
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
