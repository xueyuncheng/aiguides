package table

import "gorm.io/gorm"

type User struct {
	gorm.Model

	GoogleUserID   string
	GoogleEmail    string
	GoogleName     string
	Picture        string // Original URL from Google
	AvatarData     []byte // Stored avatar image bytes (up to 5MB). NOTE: Storing images in the database increases backup sizes and can impact performance. For large deployments, consider using external storage (e.g., filesystem, S3) and storing only the path here.
	AvatarMimeType string // MIME type of the stored avatar (e.g., "image/jpeg", "image/png")
}

type SessionMeta struct {
	gorm.Model

	SessionID string
	Title     string
}

// EmailServerConfig 邮件服务器配置
type EmailServerConfig struct {
	gorm.Model

	UserID   uint   // 关联的用户 ID
	Server   string `gorm:"not null"` // IMAP 服务器地址，例如: imap.gmail.com:993
	Username string `gorm:"not null"` // 邮箱账号
	Password string `gorm:"not null"` // 邮箱密码或应用专用密码（应加密存储）
	Mailbox  string // 邮箱文件夹名称，默认为 INBOX
	Name     string // 配置名称，用于用户识别多个邮箱
	IsDefault bool  `gorm:"default:false"` // 是否为默认邮箱
}

// GetAllModels 获取所有已注册的数据库模型
func GetAllModels() []any {
	return []any{
		&User{},
		&SessionMeta{},
		&EmailServerConfig{},
	}
}
