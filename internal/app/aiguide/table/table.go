package table

import (
	"time"

	"aiguide/internal/pkg/constant"
)

// Model 自定义模型，与 gorm.Model 类似但 ID 使用 int 类型
type Model struct {
	ID        int `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

type User struct {
	Model

	GoogleUserID   string
	GoogleEmail    string
	GoogleName     string
	Picture        string // Original URL from Google
	AvatarData     []byte // Stored avatar image bytes (up to 5MB). NOTE: Storing images in the database increases backup sizes and can impact performance. For large deployments, consider using external storage (e.g., filesystem, S3) and storing only the path here.
	AvatarMimeType string // MIME type of the stored avatar (e.g., "image/jpeg", "image/png")
}

type SessionMeta struct {
	Model

	SessionID string
	Title     string
}

// EmailServerConfig 邮件服务器配置
type EmailServerConfig struct {
	Model

	UserID    int    // 关联的用户 ID
	Server    string `gorm:"not null"` // IMAP 服务器地址，例如: imap.gmail.com:993
	Username  string `gorm:"not null"` // 邮箱账号
	Password  string `gorm:"not null"` // 邮箱密码或应用专用密码（应加密存储）
	Mailbox   string // 邮箱文件夹名称，默认为 INBOX
	Name      string // 配置名称，用于用户识别多个邮箱
	IsDefault bool   `gorm:"default:false"` // 是否为默认邮箱
}

// UserMemory 用户记忆，用于跨会话记住用户特征
type UserMemory struct {
	Model

	UserID     int                `gorm:"not null;index"`         // 关联的用户 ID
	MemoryType constant.MemoryType `gorm:"not null;index"`         // 记忆类型：fact(事实), preference(偏好), context(上下文)
	Content    string              `gorm:"not null;type:text"`     // 记忆内容
	Importance int                 `gorm:"default:5"`              // 重要性（1-10），用于后续优先级排序
	Metadata   string              `gorm:"type:text"`              // 额外的元数据（JSON格式）
}

// Task represents a subtask in a plan
type Task struct {
	Model

	SessionID   string                `gorm:"index" json:"session_id"`
	ParentID    int                   `json:"parent_id"` // 父任务ID，支持层级关系，0 表示无父任务
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      constant.TaskStatus   `gorm:"type:varchar(20);default:pending" json:"status"` // pending, in_progress, completed, failed
	DependsOn   string                `json:"depends_on"`                                     // JSON array of task IDs
	Priority    constant.TaskPriority `gorm:"type:int;default:0" json:"priority"`             // 0=low, 1=medium, 2=high
	Result      string                `json:"result,omitempty"`                               // 任务执行结果
}

// GetAllModels 获取所有已注册的数据库模型
func GetAllModels() []any {
	return []any{
		&User{},
		&SessionMeta{},
		&EmailServerConfig{},
		&UserMemory{},
		&Task{},
	}
}
