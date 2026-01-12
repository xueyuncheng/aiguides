package setting

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// EmailServerConfigRequest 邮件服务器配置请求
type EmailServerConfigRequest struct {
	Server    string `json:"server" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Mailbox   string `json:"mailbox"`
	Name      string `json:"name" binding:"required"`
	IsDefault bool   `json:"is_default"`
}

// EmailServerConfigResponse 邮件服务器配置响应
type EmailServerConfigResponse struct {
	ID        uint      `json:"id"`
	Server    string    `json:"server"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Mailbox   string    `json:"mailbox"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateEmailServerConfig 创建邮件服务器配置
func (s *Setting) CreateEmailServerConfig(c *gin.Context) {
	var req EmailServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("c.ShouldBindJSON() error", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	googleUserID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := s.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		slog.Error("db.First() error", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	// 设置默认邮箱文件夹
	if req.Mailbox == "" {
		req.Mailbox = "INBOX"
	}

	config := table.EmailServerConfig{
		UserID:    user.ID,
		Server:    req.Server,
		Username:  req.Username,
		Password:  req.Password, // SECURITY WARNING: Stored in plain text. Should be encrypted in production.
		Mailbox:   req.Mailbox,
		Name:      req.Name,
		IsDefault: req.IsDefault,
	}

	if err := s.db.Create(&config).Error; err != nil {
		slog.Error("db.Create() error", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败: " + err.Error()})
		return
	}

	resp := EmailServerConfigResponse{
		ID:        config.ID,
		Server:    config.Server,
		Username:  config.Username,
		Password:  config.Password,
		Mailbox:   config.Mailbox,
		Name:      config.Name,
		IsDefault: config.IsDefault,
		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
	}

	c.JSON(http.StatusCreated, resp)
}

// ListEmailServerConfigs 列出所有邮件服务器配置
func (s *Setting) ListEmailServerConfigs(c *gin.Context) {
	googleUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := s.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	var configs []table.EmailServerConfig
	if err := s.db.Where("user_id = ?", user.ID).Order("is_default DESC, created_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询配置失败: " + err.Error()})
		return
	}

	response := make([]EmailServerConfigResponse, 0, len(configs))
	for i := range configs {
		cfg := configs[i]
		item := EmailServerConfigResponse{
			ID:        cfg.ID,
			Server:    cfg.Server,
			Username:  cfg.Username,
			Password:  cfg.Password,
			Mailbox:   cfg.Mailbox,
			Name:      cfg.Name,
			IsDefault: cfg.IsDefault,
			CreatedAt: cfg.CreatedAt,
			UpdatedAt: cfg.UpdatedAt,
		}
		response = append(response, item)
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// GetEmailServerConfig 获取指定邮件服务器配置
func (s *Setting) GetEmailServerConfig(c *gin.Context) {
	googleUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := s.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var config table.EmailServerConfig
	if err := s.db.Where("id = ? AND user_id = ?", id, user.ID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	resp := EmailServerConfigResponse{
		ID:        config.ID,
		Server:    config.Server,
		Username:  config.Username,
		Password:  config.Password,
		Mailbox:   config.Mailbox,
		Name:      config.Name,
		IsDefault: config.IsDefault,
		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateEmailServerConfig 更新邮件服务器配置
func (s *Setting) UpdateEmailServerConfig(c *gin.Context) {
	var req EmailServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	googleUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := s.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var config table.EmailServerConfig
	if err := s.db.Where("id = ? AND user_id = ?", id, user.ID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	// 如果设置为默认，取消其他配置的默认状态
	if req.IsDefault && !config.IsDefault {
		if err := s.db.Model(&table.EmailServerConfig{}).
			Where("user_id = ? AND id != ?", user.ID, id).
			Update("is_default", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新默认配置失败"})
			return
		}
	}

	// 设置默认邮箱文件夹
	if req.Mailbox == "" {
		req.Mailbox = "INBOX"
	}

	// 更新配置
	config.Server = req.Server
	config.Username = req.Username
	if req.Password != "" {
		// Only update password if provided (SECURITY WARNING: Stored in plain text)
		config.Password = req.Password
	}
	config.Mailbox = req.Mailbox
	config.Name = req.Name
	config.IsDefault = req.IsDefault

	if err := s.db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败: " + err.Error()})
		return
	}

	resp := EmailServerConfigResponse{
		ID:        config.ID,
		Server:    config.Server,
		Username:  config.Username,
		Password:  config.Password,
		Mailbox:   config.Mailbox,
		Name:      config.Name,
		IsDefault: config.IsDefault,
		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteEmailServerConfig 删除邮件服务器配置
func (s *Setting) DeleteEmailServerConfig(c *gin.Context) {
	googleUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := s.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	result := s.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&table.EmailServerConfig{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除配置失败: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
