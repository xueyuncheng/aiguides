package constant

const (
	ContextKeyTx              string = "tx"
	ContextKeySessionID       string = "session_id"
	ContextKeyUserID          string = "user_id"
	ContextKeyGoogleUserID    string = "google_user_id"
	ContextKeyGoogleUserEmail string = "google_user_email"
	ContextKeyUserName        string = "user_name"
)

type AppName string

func (a AppName) String() string {
	return string(a)
}

const (
	AppNameAssistant AppName = "assistant"
)

// TaskStatus 任务状态类型
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// Valid 检查任务状态是否有效
func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusCompleted, TaskStatusFailed:
		return true
	}
	return false
}

// TaskPriority 任务优先级类型
type TaskPriority int

const (
	TaskPriorityLow    TaskPriority = 0
	TaskPriorityMedium TaskPriority = 1
	TaskPriorityHigh   TaskPriority = 2
)

// Valid 检查任务优先级是否有效
func (p TaskPriority) Valid() bool {
	return p >= TaskPriorityLow && p <= TaskPriorityHigh
}

// String 返回优先级的字符串表示
func (p TaskPriority) String() string {
	switch p {
	case TaskPriorityLow:
		return "low"
	case TaskPriorityMedium:
		return "medium"
	case TaskPriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}
