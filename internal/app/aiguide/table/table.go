package table

import "gorm.io/gorm"

type User struct {
	gorm.Model

	GoogleUserID   string
	GoogleEmail    string
	GoogleName     string
	Picture        string // Original URL from Google
	AvatarData     []byte // Stored avatar image data
	AvatarMimeType string // MIME type of the stored avatar (e.g., "image/jpeg", "image/png")
}

type SessionMeta struct {
	gorm.Model

	SessionID string
	Title     string
}

// GetAllModels 获取所有已注册的数据库模型
func GetAllModels() []any {
	return []any{
		&User{},
		&SessionMeta{},
	}
}
