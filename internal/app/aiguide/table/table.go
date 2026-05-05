package table

import (
	"time"

	"aiguide/internal/pkg/constant"
)

// Model 自定义模型，与 gorm.Model 类似但 ID 使用 int 类型
type Model struct {
	ID        int        `gorm:"column:id;primarykey"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

type User struct {
	Model

	GoogleUserID            string `gorm:"column:google_user_id"`
	GoogleEmail             string `gorm:"column:google_email"`
	GoogleName              string `gorm:"column:google_name"`
	Picture                 string `gorm:"column:picture"` // Original URL from Google
	AvatarData              []byte `gorm:"column:avatar_data"` // Stored avatar image bytes (up to 5MB). NOTE: Storing images in the database increases backup sizes and can impact performance. For large deployments, consider using external storage (e.g., filesystem, S3) and storing only the path here.
	AvatarMimeType          string `gorm:"column:avatar_mime_type"` // MIME type of the stored avatar (e.g., "image/jpeg", "image/png")
	GoogleOAuthRefreshToken string `gorm:"column:google_oauth_refresh_token"` // Google OAuth refresh token for Calendar API access (plain text; encrypt in production)
}

type SessionMeta struct {
	Model

	SessionID           string `gorm:"column:session_id;uniqueIndex"`
	Title               string `gorm:"column:title"`
	ThreadID            string `gorm:"column:thread_id;index"`
	ProjectID           int    `gorm:"column:project_id;not null;default:0;index"`
	Version             int    `gorm:"column:version;default:1"`
	ParentSessionID     string `gorm:"column:parent_session_id"`
	EditedFromMessageID string `gorm:"column:edited_from_message_id"`
}

type Project struct {
	Model

	UserID int    `gorm:"column:user_id;not null;uniqueIndex:idx_project_user_name"`
	Name   string `gorm:"column:name;not null;uniqueIndex:idx_project_user_name"`
}

// EmailServerConfig 邮件服务器配置
type EmailServerConfig struct {
	Model

	UserID     int    `gorm:"column:user_id"` // 关联的用户 ID
	Server     string `gorm:"column:server;not null"` // IMAP 服务器地址，例如: imap.gmail.com:993
	SMTPServer string `gorm:"column:smtp_server"` // SMTP 服务器地址，例如: smtp.gmail.com:587
	Username   string `gorm:"column:username;not null"` // 邮箱账号
	Password   string `gorm:"column:password;not null"` // 邮箱密码或应用专用密码（应加密存储）
	Mailbox    string `gorm:"column:mailbox"` // 邮箱文件夹名称，默认为 INBOX
	Name       string `gorm:"column:name"` // 配置名称，用于用户识别多个邮箱
	IsDefault  bool   `gorm:"column:is_default;default:false"` // 是否为默认邮箱
}

// SSHAuthMethod distinguishes password-based from key-based SSH authentication.
type SSHAuthMethod string

const (
	SSHAuthMethodPassword SSHAuthMethod = "password"
	SSHAuthMethodKey      SSHAuthMethod = "key"
)

// SSHServerConfig SSH server configuration for a user
type SSHServerConfig struct {
	Model

	UserID     int           `gorm:"column:user_id;not null;index"`                // Associated user ID
	Name       string        `gorm:"column:name;not null"`                         // Display name to identify the server
	Host       string        `gorm:"column:host;not null"`                         // Hostname or IP, e.g. 192.168.1.10
	Port       int           `gorm:"column:port;not null;default:22"`              // SSH port, default 22
	Username   string        `gorm:"column:username;not null"`                     // SSH login username
	AuthMethod SSHAuthMethod `gorm:"column:auth_method;not null;default:'password'"` // "password" or "key"
	Password   string        `gorm:"column:password;not null;default:''"`          // Used when AuthMethod == "password" (plain text; encrypt in production)
	PrivateKey string        `gorm:"column:private_key;not null;default:'';type:text"` // PEM-encoded private key; used when AuthMethod == "key"
	Passphrase string        `gorm:"column:passphrase;not null;default:''"`        // Optional passphrase for encrypted private keys
	IsDefault  bool          `gorm:"column:is_default;default:false"`              // Whether this is the user's default SSH server
}

// UserMemory 用户记忆，用于跨会话记住用户特征
type UserMemory struct {
	Model

	UserID     int                 `gorm:"column:user_id;not null;index"`      // 关联的用户 ID
	MemoryType constant.MemoryType `gorm:"column:memory_type;not null;index"`  // 记忆类型：fact(事实), preference(偏好), context(上下文)
	Content    string              `gorm:"column:content;not null;type:text"`  // 记忆内容
	Importance int                 `gorm:"column:importance;default:5"`        // 重要性（1-10），用于后续优先级排序
	Metadata   string              `gorm:"column:metadata;type:text"`          // 额外的元数据（JSON格式）
}

// Task represents a subtask in a plan
type Task struct {
	Model

	SessionID   string                `gorm:"column:session_id;index" json:"session_id"`
	ParentID    int                   `gorm:"column:parent_id" json:"parent_id"` // 父任务ID，支持层级关系，0 表示无父任务
	Title       string                `gorm:"column:title" json:"title"`
	Description string                `gorm:"column:description" json:"description"`
	Status      constant.TaskStatus   `gorm:"column:status;type:varchar(20);default:pending" json:"status"` // pending, in_progress, completed, failed
	DependsOn   string                `gorm:"column:depends_on" json:"depends_on"`                          // JSON array of task IDs
	Priority    constant.TaskPriority `gorm:"column:priority;type:int;default:0" json:"priority"`           // 0=low, 1=medium, 2=high
	Result      string                `gorm:"column:result" json:"result,omitempty"`                        // 任务执行结果
}

// ScheduledTask represents a user-defined timed task.
type ScheduledTask struct {
	Model

	UserID       int        `gorm:"column:user_id;not null;index" json:"user_id"`
	SessionID    string     `gorm:"column:session_id;index" json:"session_id"`
	Title        string     `gorm:"column:title;not null" json:"title"`
	Action       string     `gorm:"column:action;type:text;not null" json:"action"`
	ScheduleType string     `gorm:"column:schedule_type;type:varchar(20);not null;index" json:"schedule_type"` // once, daily, weekly
	RunAt        string     `gorm:"column:run_at;not null" json:"run_at"`                                      // once=RFC3339, daily/weekly=HH:MM
	Weekday      int        `gorm:"column:weekday" json:"weekday"`                                             // only for weekly: 0(Sun)-6(Sat)
	Timezone     string     `gorm:"column:timezone;not null" json:"timezone"`
	TargetEmail  string     `gorm:"column:target_email" json:"target_email,omitempty"`
	Enabled      bool       `gorm:"column:enabled;default:true;index" json:"enabled"`
	LastRunAt    *time.Time `gorm:"column:last_run_at" json:"last_run_at,omitempty"`
	NextRunAt    time.Time  `gorm:"column:next_run_at;not null;index" json:"next_run_at"`
}

// SharedConversation represents a shared conversation link
type SharedConversation struct {
	Model

	ShareID    string    `gorm:"column:share_id;uniqueIndex;not null"` // UUID for the shareable link
	SessionID  string    `gorm:"column:session_id;not null;index"`     // Original session ID
	UserID     int       `gorm:"column:user_id;not null;index"`        // User who created the share
	AppName    string    `gorm:"column:app_name;not null"`             // Agent/app name
	ExpiresAt  time.Time `gorm:"column:expires_at;not null;index"`     // Link expiration time
	AccessedAt time.Time `gorm:"column:accessed_at"`                   // Last access time (optional tracking)
}

// FileAsset represents a user-owned file stored on disk.
type FileAsset struct {
	Model

	UserID       int                           `gorm:"column:user_id;not null;index"`
	SessionID    string                        `gorm:"column:session_id;not null;default:'';index"`
	Kind         constant.FileAssetKind        `gorm:"column:kind;type:varchar(20);not null;index"`
	MimeType     string                        `gorm:"column:mime_type;not null;default:''"`
	OriginalName string                        `gorm:"column:original_name;not null;default:''"`
	StoragePath  string                        `gorm:"column:storage_path;not null;default:'';uniqueIndex"`
	SizeBytes    int64                         `gorm:"column:size_bytes;not null;default:0"`
	SHA256       string                        `gorm:"column:sha256;not null;default:'';index"`
	Status       constant.FileAssetStatus      `gorm:"column:status;type:varchar(20);not null;default:ready;index"`
	TextStatus   constant.PDFTextExtractStatus `gorm:"column:text_status;type:varchar(20);not null;default:pending;index"`
	TextPages    int                           `gorm:"column:text_pages;not null;default:0"`
	TextChars    int                           `gorm:"column:text_chars;not null;default:0"`
	TextError    string                        `gorm:"column:text_error;type:text;not null;default:''"`
}

// PDFTextPage stores extracted plain text for a single page in a PDF asset.
type PDFTextPage struct {
	Model

	FileID         int    `gorm:"column:file_id;not null;index:idx_pdf_text_page_file_page,unique"`
	UserID         int    `gorm:"column:user_id;not null;index"`
	SessionID      string `gorm:"column:session_id;not null;default:'';index"`
	PageNumber     int    `gorm:"column:page_number;not null;index:idx_pdf_text_page_file_page,unique"`
	CharacterCount int    `gorm:"column:character_count;not null;default:0"`
	Text           string `gorm:"column:text;type:text;not null;default:''"`
}

// PDFJob tracks an asynchronous PDF processing task.
type PDFJob struct {
	Model

	UserID        int                   `gorm:"column:user_id;not null;index"`
	SessionID     string                `gorm:"column:session_id;not null;index"`
	Type          constant.PDFJobType   `gorm:"column:type;type:varchar(40);not null;index"`
	Status        constant.PDFJobStatus `gorm:"column:status;type:varchar(20);not null;default:pending;index"`
	InputFileIDs  string                `gorm:"column:input_file_ids;type:text;not null;default:''"`
	OutputFileID  int                   `gorm:"column:output_file_id;not null;default:0;index"`
	Params        string                `gorm:"column:params;type:text;not null;default:''"`
	ResultSummary string                `gorm:"column:result_summary;type:text;not null;default:''"`
	ErrorMessage  string                `gorm:"column:error_message;type:text;not null;default:''"`
	StartedAt     time.Time             `gorm:"column:started_at;not null"`
	CompletedAt   time.Time             `gorm:"column:completed_at;not null"`
}

// AudioJob tracks an audio transcription task.
type AudioJob struct {
	Model

	UserID          int                     `gorm:"column:user_id;not null;index"`
	SessionID       string                  `gorm:"column:session_id;not null;index"`
	FileID          int                     `gorm:"column:file_id;not null;index"`
	Status          constant.AudioJobStatus `gorm:"column:status;type:varchar(20);not null;default:pending;index"`
	ModelName       string                  `gorm:"column:model_name;not null;default:''"`
	MimeType        string                  `gorm:"column:mime_type;not null;default:''"`
	Prompt          string                  `gorm:"column:prompt;type:text;not null;default:''"`
	DurationMs      int64                   `gorm:"column:duration_ms;not null;default:0"`
	ChunkCount      int                     `gorm:"column:chunk_count;not null;default:0"`
	Language        string                  `gorm:"column:language;not null;default:''"`
	TranscriptText  string                  `gorm:"column:transcript_text;type:text;not null;default:''"`
	TranscriptChars int                     `gorm:"column:transcript_chars;not null;default:0"`
	ErrorMessage    string                  `gorm:"column:error_message;type:text;not null;default:''"`
	StartedAt       time.Time               `gorm:"column:started_at;not null"`
	CompletedAt     time.Time               `gorm:"column:completed_at;not null"`
}

// AudioTranscriptChunk stores the transcript result for a chunk of an audio file.
type AudioTranscriptChunk struct {
	Model

	JobID           int                                 `gorm:"column:job_id;not null;index"`
	ChunkIndex      int                                 `gorm:"column:chunk_index;not null;index"`
	StartMs         int64                               `gorm:"column:start_ms;not null;default:0"`
	EndMs           int64                               `gorm:"column:end_ms;not null;default:0"`
	OverlapStartMs  int64                               `gorm:"column:overlap_start_ms;not null;default:0"`
	OverlapEndMs    int64                               `gorm:"column:overlap_end_ms;not null;default:0"`
	Status          constant.AudioTranscriptChunkStatus `gorm:"column:status;type:varchar(20);not null;default:pending;index"`
	Language        string                              `gorm:"column:language;not null;default:''"`
	TranscriptText  string                              `gorm:"column:transcript_text;type:text;not null;default:''"`
	TranscriptChars int                                 `gorm:"column:transcript_chars;not null;default:0"`
	ErrorMessage    string                              `gorm:"column:error_message;type:text;not null;default:''"`
}

// GetAllModels 获取所有已注册的数据库模型
func GetAllModels() []any {
	return []any{
		&User{},
		&SessionMeta{},
		&Project{},
		&EmailServerConfig{},
		&SSHServerConfig{},
		&UserMemory{},
		&Task{},
		&ScheduledTask{},
		&SharedConversation{},
		&FileAsset{},
		&PDFTextPage{},
		&PDFJob{},
		&AudioJob{},
		&AudioTranscriptChunk{},
	}
}
