package table

import "gorm.io/gorm"

type User struct {
	gorm.Model

	GoogleUserID string
	GoogleEmail  string
	GoogleName   string
	Picture      string
}

type SessionMeta struct {
	gorm.Model

	SessionID string
	Title     string
}
