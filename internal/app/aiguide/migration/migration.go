package migration

import (
	"aiguide/internal/app/aiguide/table"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

type Migrator struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Migrator {
	return &Migrator{
		db: db,
	}
}

func (m *Migrator) Run() error {
	models := table.GetAllModels()
	if err := m.db.AutoMigrate(models...); err != nil {
		slog.Error("failed to run database auto migration", "err", err)
		return fmt.Errorf("failed to run database auto migration: %w", err)
	}
	return nil
}
