package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"net/http"
	"strconv"

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
	ID        uint   `json:"id"`
	Server    string `json:"server"`
	Username  string `json:"username"`
	Mailbox   string `json:"mailbox"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateEmailServer 创建邮件服务器配置
func (a *AIGuide) CreateEmailServer(c *gin.Context) {
	googleUserID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := a.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	var req EmailServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 如果设置为默认，取消其他配置的默认状态
	if req.IsDefault {
		if err := a.db.Model(&table.EmailServerConfig{}).
			Where("user_id = ?", user.ID).
			Update("is_default", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新默认配置失败"})
			return
		}
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

	if err := a.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toEmailServerConfigResponse(&config))
}

// ListEmailServers 列出所有邮件服务器配置
func (a *AIGuide) ListEmailServers(c *gin.Context) {
	googleUserID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := a.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	var configs []table.EmailServerConfig
	if err := a.db.Where("user_id = ?", user.ID).Order("is_default DESC, created_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询配置失败: " + err.Error()})
		return
	}

	response := make([]EmailServerConfigResponse, 0, len(configs))
	for i := range configs {
		response = append(response, toEmailServerConfigResponse(&configs[i]))
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// GetEmailServer 获取指定邮件服务器配置
func (a *AIGuide) GetEmailServer(c *gin.Context) {
	googleUserID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := a.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var config table.EmailServerConfig
	if err := a.db.Where("id = ? AND user_id = ?", id, user.ID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	c.JSON(http.StatusOK, toEmailServerConfigResponse(&config))
}

// UpdateEmailServer 更新邮件服务器配置
func (a *AIGuide) UpdateEmailServer(c *gin.Context) {
	googleUserID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := a.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var req EmailServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	var config table.EmailServerConfig
	if err := a.db.Where("id = ? AND user_id = ?", id, user.ID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	// 如果设置为默认，取消其他配置的默认状态
	if req.IsDefault && !config.IsDefault {
		if err := a.db.Model(&table.EmailServerConfig{}).
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

	if err := a.db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, toEmailServerConfigResponse(&config))
}

// DeleteEmailServer 删除邮件服务器配置
func (a *AIGuide) DeleteEmailServer(c *gin.Context) {
	googleUserID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查找用户
	var user table.User
	if err := a.db.Where("google_user_id = ?", googleUserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	result := a.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&table.EmailServerConfig{})
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

// toEmailServerConfigResponse 将数据库模型转换为响应格式
func toEmailServerConfigResponse(config *table.EmailServerConfig) EmailServerConfigResponse {
	return EmailServerConfigResponse{
		ID:        config.ID,
		Server:    config.Server,
		Username:  config.Username,
		Mailbox:   config.Mailbox,
		Name:      config.Name,
		IsDefault: config.IsDefault,
		CreatedAt: config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
