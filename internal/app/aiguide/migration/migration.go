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
		slog.Error("m.db.AutoMigrate() error", "err", err)
		return fmt.Errorf("m.db.AutoMigrate() error, err = %w", err)
	}
	return nil
}
