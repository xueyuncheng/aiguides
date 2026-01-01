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

func New(dialector gorm.Dialector) (*Migrator, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		slog.Error("gorm.Open() error", "err", err)
		return nil, fmt.Errorf("gorm.Open() error, err = %w", err)
	}

	migrator := &Migrator{
		db: db,
	}

	return migrator, nil
}

func (m *Migrator) Run() error {
	models := table.GetAllModels()
	if err := m.db.AutoMigrate(models...); err != nil {
		slog.Error("m.db.AutoMigrate() error", "err", err)
		return fmt.Errorf("m.db.AutoMigrate() error, err = %w", err)
	}
	return nil
}
