package tools

import (
	"aiguide/internal/app/aiguide/table"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"gorm.io/gorm"
)

// MemoryInput 定义记忆工具的输入参数
type MemoryInput struct {
	Action     string `json:"action" jsonschema:"操作类型：save(保存), retrieve(检索), update(更新), delete(删除)"`
	MemoryType string `json:"memory_type,omitempty" jsonschema:"记忆类型：fact(事实), preference(偏好), context(上下文)"`
	Content    string `json:"content,omitempty" jsonschema:"记忆内容"`
	MemoryID   int    `json:"memory_id,omitempty" jsonschema:"记忆ID"`
	Importance int    `json:"importance,omitempty" jsonschema:"重要性（1-10），默认为5"`
	UserID     int    `json:"user_id" jsonschema:"用户ID"`
}

// MemoryOutput 定义记忆工具的输出结果
type MemoryOutput struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message"`
	Memories []MemoryItem      `json:"memories,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MemoryItem 记忆项
type MemoryItem struct {
	ID         int    `json:"id"`
	MemoryType string `json:"memory_type"`
	Content    string `json:"content"`
	Importance int    `json:"importance"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// memoryHandler 实现记忆工具的核心逻辑
type memoryHandler struct {
	db *gorm.DB
}

// NewMemoryTool 创建新的记忆工具实例
func NewMemoryTool(db *gorm.DB) (tool.Tool, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	handler := &memoryHandler{db: db}

	config := functiontool.Config{
		Name:        "manage_memory",
		Description: "管理用户记忆的工具。可以保存、检索、更新和删除用户的记忆信息。记忆会跨会话保持，让AI能够记住用户的特征、偏好和上下文。",
	}

	handlerFunc := func(ctx tool.Context, input *MemoryInput) (*MemoryOutput, error) {
		return handler.handleMemory(input)
	}

	return functiontool.New(config, handlerFunc)
}

// handleMemory 处理记忆操作请求
func (h *memoryHandler) handleMemory(input *MemoryInput) (*MemoryOutput, error) {
	slog.Debug("Memory tool called", "action", input.Action, "user_id", input.UserID)

	if input.UserID == 0 {
		return &MemoryOutput{
			Success: false,
			Error:   "user_id is required",
		}, nil
	}

	switch input.Action {
	case "save":
		return h.saveMemory(input)
	case "retrieve":
		return h.retrieveMemories(input)
	case "update":
		return h.updateMemory(input)
	case "delete":
		return h.deleteMemory(input)
	default:
		return &MemoryOutput{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s. Valid actions: save, retrieve, update, delete", input.Action),
		}, nil
	}
}

// saveMemory 保存新的记忆
func (h *memoryHandler) saveMemory(input *MemoryInput) (*MemoryOutput, error) {
	if input.Content == "" {
		return &MemoryOutput{
			Success: false,
			Error:   "content is required for save action",
		}, nil
	}

	if input.MemoryType == "" {
		input.MemoryType = "fact" // 默认类型
	}

	// 验证记忆类型
	validTypes := map[string]bool{"fact": true, "preference": true, "context": true}
	if !validTypes[input.MemoryType] {
		return &MemoryOutput{
			Success: false,
			Error:   "invalid memory_type. Valid types: fact, preference, context",
		}, nil
	}

	importance := input.Importance
	if importance == 0 {
		importance = 5 // 默认重要性
	}
	if importance < 1 || importance > 10 {
		importance = 5
	}

	memory := table.UserMemory{
		UserID:     input.UserID,
		MemoryType: input.MemoryType,
		Content:    input.Content,
		Importance: importance,
	}

	if err := h.db.Create(&memory).Error; err != nil {
		slog.Error("Failed to save memory", "err", err)
		return &MemoryOutput{
			Success: false,
			Error:   "failed to save memory to database",
		}, nil
	}

	return &MemoryOutput{
		Success: true,
		Message: fmt.Sprintf("Memory saved successfully with ID %d", memory.ID),
		Memories: []MemoryItem{
			{
				ID:         memory.ID,
				MemoryType: memory.MemoryType,
				Content:    memory.Content,
				Importance: memory.Importance,
				CreatedAt:  memory.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:  memory.UpdatedAt.Format("2006-01-02 15:04:05"),
			},
		},
	}, nil
}

// retrieveMemories 检索用户的记忆
func (h *memoryHandler) retrieveMemories(input *MemoryInput) (*MemoryOutput, error) {
	query := h.db.Where("user_id = ?", input.UserID)

	// 可选：按类型过滤
	if input.MemoryType != "" {
		query = query.Where("memory_type = ?", input.MemoryType)
	}

	var memories []table.UserMemory
	// 按重要性和更新时间排序
	if err := query.Order("importance DESC, updated_at DESC").Find(&memories).Error; err != nil {
		slog.Error("Failed to retrieve memories", "err", err)
		return &MemoryOutput{
			Success: false,
			Error:   "failed to retrieve memories from database",
		}, nil
	}

	items := make([]MemoryItem, 0, len(memories))
	for _, mem := range memories {
		items = append(items, MemoryItem{
			ID:         mem.ID,
			MemoryType: mem.MemoryType,
			Content:    mem.Content,
			Importance: mem.Importance,
			CreatedAt:  mem.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  mem.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	message := fmt.Sprintf("Retrieved %d memories", len(items))
	if input.MemoryType != "" {
		message += fmt.Sprintf(" of type '%s'", input.MemoryType)
	}

	return &MemoryOutput{
		Success:  true,
		Message:  message,
		Memories: items,
	}, nil
}

// updateMemory 更新现有记忆
func (h *memoryHandler) updateMemory(input *MemoryInput) (*MemoryOutput, error) {
	if input.MemoryID == 0 {
		return &MemoryOutput{
			Success: false,
			Error:   "memory_id is required for update action",
		}, nil
	}

	if input.Content == "" {
		return &MemoryOutput{
			Success: false,
			Error:   "content is required for update action",
		}, nil
	}

	var memory table.UserMemory
	if err := h.db.Where("id = ? AND user_id = ?", input.MemoryID, input.UserID).First(&memory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &MemoryOutput{
				Success: false,
				Error:   fmt.Sprintf("memory with ID %d not found", input.MemoryID),
			}, nil
		}
		slog.Error("Failed to find memory", "err", err)
		return &MemoryOutput{
			Success: false,
			Error:   "failed to find memory in database",
		}, nil
	}

	// 更新字段
	memory.Content = input.Content
	if input.MemoryType != "" {
		validTypes := map[string]bool{"fact": true, "preference": true, "context": true}
		if validTypes[input.MemoryType] {
			memory.MemoryType = input.MemoryType
		}
	}
	if input.Importance > 0 && input.Importance <= 10 {
		memory.Importance = input.Importance
	}

	if err := h.db.Save(&memory).Error; err != nil {
		slog.Error("Failed to update memory", "err", err)
		return &MemoryOutput{
			Success: false,
			Error:   "failed to update memory in database",
		}, nil
	}

	return &MemoryOutput{
		Success: true,
		Message: fmt.Sprintf("Memory %d updated successfully", memory.ID),
		Memories: []MemoryItem{
			{
				ID:         memory.ID,
				MemoryType: memory.MemoryType,
				Content:    memory.Content,
				Importance: memory.Importance,
				CreatedAt:  memory.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:  memory.UpdatedAt.Format("2006-01-02 15:04:05"),
			},
		},
	}, nil
}

// deleteMemory 删除记忆
func (h *memoryHandler) deleteMemory(input *MemoryInput) (*MemoryOutput, error) {
	if input.MemoryID == 0 {
		return &MemoryOutput{
			Success: false,
			Error:   "memory_id is required for delete action",
		}, nil
	}

	result := h.db.Where("id = ? AND user_id = ?", input.MemoryID, input.UserID).Delete(&table.UserMemory{})
	if result.Error != nil {
		slog.Error("Failed to delete memory", "err", result.Error)
		return &MemoryOutput{
			Success: false,
			Error:   "failed to delete memory from database",
		}, nil
	}

	if result.RowsAffected == 0 {
		return &MemoryOutput{
			Success: false,
			Error:   fmt.Sprintf("memory with ID %d not found", input.MemoryID),
		}, nil
	}

	return &MemoryOutput{
		Success: true,
		Message: fmt.Sprintf("Memory %d deleted successfully", input.MemoryID),
	}, nil
}

// GetUserMemoriesAsContext 获取用户记忆并格式化为上下文文本
// 这个函数用于在聊天开始时将记忆注入到上下文中
func GetUserMemoriesAsContext(db *gorm.DB, userID int) (string, error) {
	var memories []table.UserMemory
	if err := db.Where("user_id = ?", userID).
		Order("importance DESC, updated_at DESC").
		Limit(20). // 限制数量以避免上下文过长
		Find(&memories).Error; err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "", nil
	}

	// 按类型分组
	factMemories := []string{}
	prefMemories := []string{}
	contextMemories := []string{}

	for _, mem := range memories {
		switch mem.MemoryType {
		case "fact":
			factMemories = append(factMemories, mem.Content)
		case "preference":
			prefMemories = append(prefMemories, mem.Content)
		case "context":
			contextMemories = append(contextMemories, mem.Content)
		}
	}

	// 构建上下文文本
	contextText := "## 用户记忆信息\n\n"

	if len(factMemories) > 0 {
		contextText += "**关于用户的事实：**\n"
		for _, fact := range factMemories {
			contextText += fmt.Sprintf("- %s\n", fact)
		}
		contextText += "\n"
	}

	if len(prefMemories) > 0 {
		contextText += "**用户偏好：**\n"
		for _, pref := range prefMemories {
			contextText += fmt.Sprintf("- %s\n", pref)
		}
		contextText += "\n"
	}

	if len(contextMemories) > 0 {
		contextText += "**相关上下文：**\n"
		for _, ctx := range contextMemories {
			contextText += fmt.Sprintf("- %s\n", ctx)
		}
		contextText += "\n"
	}

	return contextText, nil
}

// SerializeMemoryOutput 序列化记忆输出为 JSON 字符串（用于调试）
func SerializeMemoryOutput(output *MemoryOutput) string {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf("error serializing output: %v", err)
	}
	return string(data)
}
