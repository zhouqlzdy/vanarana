package store

import (
	"vanarana/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.Repository{},
		&model.PipelineRun{},
		&model.ModuleReport{},
		&model.JunitMetrics{},
		&model.JacocoMetrics{},
	)
}
