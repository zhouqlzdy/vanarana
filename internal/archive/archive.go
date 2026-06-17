package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create archive dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(pipelineRunID int, moduleName, reportType string, src io.Reader) (string, error) {
	filename := fmt.Sprintf("%d_%s_%s.tar.gz", pipelineRunID, moduleName, reportType)
	path := filepath.Join(s.dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("write archive file: %w", err)
	}

	return filename, nil
}

func (s *Store) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

func (s *Store) Open(filename string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.dir, filename))
}
