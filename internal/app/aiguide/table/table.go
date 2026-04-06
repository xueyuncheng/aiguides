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

	SessionID           string `gorm:"uniqueIndex"`
	Title               string
	ThreadID            string `gorm:"index"`
	ProjectID           int    `gorm:"not null;default:0;index"`
	Version             int    `gorm:"default:1"`
	ParentSessionID     string
	EditedFromMessageID string
}

type Project struct {
	Model

	UserID int    `gorm:"not null;uniqueIndex:idx_project_user_name"`
	Name   string `gorm:"not null;uniqueIndex:idx_project_user_name"`
}

// EmailServerConfig 邮件服务器配置
type EmailServerConfig struct {
	Model

	UserID     int    // 关联的用户 ID
	Server     string `gorm:"not null"` // IMAP 服务器地址，例如: imap.gmail.com:993
	SMTPServer string // SMTP 服务器地址，例如: smtp.gmail.com:587
	Username   string `gorm:"not null"` // 邮箱账号
	Password   string `gorm:"not null"` // 邮箱密码或应用专用密码（应加密存储）
	Mailbox    string // 邮箱文件夹名称，默认为 INBOX
	Name       string // 配置名称，用于用户识别多个邮箱
	IsDefault  bool   `gorm:"default:false"` // 是否为默认邮箱
}

// UserMemory 用户记忆，用于跨会话记住用户特征
type UserMemory struct {
	Model

	UserID     int                 `gorm:"not null;index"`     // 关联的用户 ID
	MemoryType constant.MemoryType `gorm:"not null;index"`     // 记忆类型：fact(事实), preference(偏好), context(上下文)
	Content    string              `gorm:"not null;type:text"` // 记忆内容
	Importance int                 `gorm:"default:5"`          // 重要性（1-10），用于后续优先级排序
	Metadata   string              `gorm:"type:text"`          // 额外的元数据（JSON格式）
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

// SharedConversation represents a shared conversation link
type SharedConversation struct {
	Model

	ShareID    string    `gorm:"uniqueIndex;not null"` // UUID for the shareable link
	SessionID  string    `gorm:"not null;index"`       // Original session ID
	UserID     int       `gorm:"not null;index"`       // User who created the share
	AppName    string    `gorm:"not null"`             // Agent/app name
	ExpiresAt  time.Time `gorm:"not null;index"`       // Link expiration time
	AccessedAt time.Time // Last access time (optional tracking)
}

// FileAsset represents a user-owned file stored on disk.
type FileAsset struct {
	Model

	UserID       int                           `gorm:"not null;index"`
	SessionID    string                        `gorm:"not null;default:'';index"`
	Kind         constant.FileAssetKind        `gorm:"type:varchar(20);not null;index"`
	MimeType     string                        `gorm:"not null;default:''"`
	OriginalName string                        `gorm:"not null;default:''"`
	StoragePath  string                        `gorm:"not null;default:'';uniqueIndex"`
	SizeBytes    int64                         `gorm:"not null;default:0"`
	SHA256       string                        `gorm:"not null;default:'';index"`
	Status       constant.FileAssetStatus      `gorm:"type:varchar(20);not null;default:ready;index"`
	TextStatus   constant.PDFTextExtractStatus `gorm:"column:text_status;type:varchar(20);not null;default:pending;index"`
	TextPages    int                           `gorm:"column:text_pages;not null;default:0"`
	TextChars    int                           `gorm:"column:text_chars;not null;default:0"`
	TextError    string                        `gorm:"column:text_error;type:text;not null;default:''"`
}

// PDFTextPage stores extracted plain text for a single page in a PDF asset.
type PDFTextPage struct {
	Model

	FileID         int    `gorm:"not null;index:idx_pdf_text_page_file_page,unique"`
	UserID         int    `gorm:"not null;index"`
	SessionID      string `gorm:"not null;default:'';index"`
	PageNumber     int    `gorm:"not null;index:idx_pdf_text_page_file_page,unique"`
	CharacterCount int    `gorm:"not null;default:0"`
	Text           string `gorm:"type:text;not null;default:''"`
}

// PDFJob tracks an asynchronous PDF processing task.
type PDFJob struct {
	Model

	UserID        int                   `gorm:"not null;index"`
	SessionID     string                `gorm:"not null;index"`
	Type          constant.PDFJobType   `gorm:"type:varchar(40);not null;index"`
	Status        constant.PDFJobStatus `gorm:"type:varchar(20);not null;default:pending;index"`
	InputFileIDs  string                `gorm:"type:text;not null;default:''"`
	OutputFileID  int                   `gorm:"not null;default:0;index"`
	Params        string                `gorm:"type:text;not null;default:''"`
	ResultSummary string                `gorm:"type:text;not null;default:''"`
	ErrorMessage  string                `gorm:"type:text;not null;default:''"`
	StartedAt     time.Time             `gorm:"not null"`
	CompletedAt   time.Time             `gorm:"not null"`
}

// AudioJob tracks an audio transcription task.
type AudioJob struct {
	Model

	UserID          int                     `gorm:"not null;index"`
	SessionID       string                  `gorm:"not null;index"`
	FileID          int                     `gorm:"not null;index"`
	Status          constant.AudioJobStatus `gorm:"type:varchar(20);not null;default:pending;index"`
	ModelName       string                  `gorm:"not null;default:''"`
	MimeType        string                  `gorm:"not null;default:''"`
	Prompt          string                  `gorm:"type:text;not null;default:''"`
	DurationMs      int64                   `gorm:"not null;default:0"`
	ChunkCount      int                     `gorm:"not null;default:0"`
	Language        string                  `gorm:"not null;default:''"`
	TranscriptText  string                  `gorm:"type:text;not null;default:''"`
	TranscriptChars int                     `gorm:"not null;default:0"`
	ErrorMessage    string                  `gorm:"type:text;not null;default:''"`
	StartedAt       time.Time               `gorm:"not null"`
	CompletedAt     time.Time               `gorm:"not null"`
}

// AudioTranscriptChunk stores the transcript result for a chunk of an audio file.
type AudioTranscriptChunk struct {
	Model

	JobID           int                                 `gorm:"not null;index"`
	ChunkIndex      int                                 `gorm:"not null;index"`
	StartMs         int64                               `gorm:"not null;default:0"`
	EndMs           int64                               `gorm:"not null;default:0"`
	OverlapStartMs  int64                               `gorm:"not null;default:0"`
	OverlapEndMs    int64                               `gorm:"not null;default:0"`
	Status          constant.AudioTranscriptChunkStatus `gorm:"type:varchar(20);not null;default:pending;index"`
	Language        string                              `gorm:"not null;default:''"`
	TranscriptText  string                              `gorm:"type:text;not null;default:''"`
	TranscriptChars int                                 `gorm:"not null;default:0"`
	ErrorMessage    string                              `gorm:"type:text;not null;default:''"`
}

// GetAllModels 获取所有已注册的数据库模型
func GetAllModels() []any {
	return []any{
		&User{},
		&SessionMeta{},
		&Project{},
		&EmailServerConfig{},
		&UserMemory{},
		&Task{},
		&SharedConversation{},
		&FileAsset{},
		&PDFTextPage{},
		&PDFJob{},
		&AudioJob{},
		&AudioTranscriptChunk{},
	}
}
