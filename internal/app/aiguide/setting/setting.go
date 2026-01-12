package setting

import "gorm.io/gorm"

type Setting struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Setting {
	s := &Setting{
		db: db,
	}

	return s
}
