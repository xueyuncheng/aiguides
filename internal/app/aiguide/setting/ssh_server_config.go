package setting

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// SSHServerConfigRequest is the request body for creating/updating an SSH server config.
type SSHServerConfigRequest struct {
	Name       string              `json:"name"        binding:"required"`
	Host       string              `json:"host"        binding:"required"`
	Port       int                 `json:"port"`
	Username   string              `json:"username"    binding:"required"`
	AuthMethod table.SSHAuthMethod `json:"auth_method"`
	Password   string              `json:"password"`
	PrivateKey string              `json:"private_key"`
	Passphrase string              `json:"passphrase"`
	IsDefault  bool                `json:"is_default"`
}

// SSHServerConfigResponse is the response body for SSH server config endpoints.
// Private credentials are intentionally omitted from list responses; the full
// private_key and passphrase are only returned by the single-item GET endpoint
// so the edit form can pre-populate them.
type SSHServerConfigResponse struct {
	ID         int                 `json:"id"`
	Name       string              `json:"name"`
	Host       string              `json:"host"`
	Port       int                 `json:"port"`
	Username   string              `json:"username"`
	AuthMethod table.SSHAuthMethod `json:"auth_method"`
	Password   string              `json:"password,omitempty"`
	PrivateKey string              `json:"private_key,omitempty"`
	Passphrase string              `json:"passphrase,omitempty"`
	IsDefault  bool                `json:"is_default"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

func toSSHResponse(cfg table.SSHServerConfig) SSHServerConfigResponse {
	return SSHServerConfigResponse{
		ID:         cfg.ID,
		Name:       cfg.Name,
		Host:       cfg.Host,
		Port:       cfg.Port,
		Username:   cfg.Username,
		AuthMethod: cfg.AuthMethod,
		Password:   cfg.Password,
		PrivateKey: cfg.PrivateKey,
		Passphrase: cfg.Passphrase,
		IsDefault:  cfg.IsDefault,
		CreatedAt:  cfg.CreatedAt,
		UpdatedAt:  cfg.UpdatedAt,
	}
}

// CreateSSHServerConfig creates a new SSH server config for the authenticated user.
func (s *Setting) CreateSSHServerConfig(c *gin.Context) {
	var req SSHServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("c.ShouldBindJSON() error", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	userID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated in CreateSSHServerConfig")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}

	authMethod := req.AuthMethod
	if authMethod == "" {
		authMethod = table.SSHAuthMethodPassword
	}

	cfg := table.SSHServerConfig{
		UserID:     userID,
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		AuthMethod: authMethod,
		Password:   req.Password, // SECURITY WARNING: stored in plain text
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
		IsDefault:  req.IsDefault,
	}

	if err := s.db.Create(&cfg).Error; err != nil {
		slog.Error("db.Create() error in CreateSSHServerConfig", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create config: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toSSHResponse(cfg))
}

// ListSSHServerConfigs returns all SSH server configs for the authenticated user.
func (s *Setting) ListSSHServerConfigs(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated in ListSSHServerConfigs")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var configs []table.SSHServerConfig
	if err := s.db.Where("user_id = ?", userID).Order("is_default DESC, created_at DESC").Find(&configs).Error; err != nil {
		slog.Error("db.Find() error in ListSSHServerConfigs", "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs: " + err.Error()})
		return
	}

	response := make([]SSHServerConfigResponse, 0, len(configs))
	for _, cfg := range configs {
		response = append(response, toSSHResponse(cfg))
	}

	c.JSON(http.StatusOK, gin.H{"configs": response})
}

// GetSSHServerConfig returns a single SSH server config by ID.
func (s *Setting) GetSSHServerConfig(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated in GetSSHServerConfig")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("strconv.Atoi() error in GetSSHServerConfig", "id", c.Param("id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var cfg table.SSHServerConfig
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&cfg).Error; err != nil {
		slog.Error("db.First() error in GetSSHServerConfig", "config_id", id, "user_id", userID, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, toSSHResponse(cfg))
}

// UpdateSSHServerConfig updates an existing SSH server config.
func (s *Setting) UpdateSSHServerConfig(c *gin.Context) {
	var req SSHServerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("c.ShouldBindJSON() error in UpdateSSHServerConfig", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	userID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated in UpdateSSHServerConfig")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("strconv.Atoi() error in UpdateSSHServerConfig", "id", c.Param("id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var cfg table.SSHServerConfig
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&cfg).Error; err != nil {
		slog.Error("db.First() error in UpdateSSHServerConfig", "config_id", id, "user_id", userID, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// If setting this one as default, clear other defaults for the user.
	if req.IsDefault && !cfg.IsDefault {
		if err := s.db.Model(&table.SSHServerConfig{}).
			Where("user_id = ? AND id != ?", userID, id).
			Update("is_default", false).Error; err != nil {
			slog.Error("db.Update() error clearing is_default in UpdateSSHServerConfig", "user_id", userID, "id", id, "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default config"})
			return
		}
	}

	if req.Port == 0 {
		req.Port = 22
	}

	authMethod := req.AuthMethod
	if authMethod == "" {
		authMethod = table.SSHAuthMethodPassword
	}

	cfg.Name = req.Name
	cfg.Host = req.Host
	cfg.Port = req.Port
	cfg.Username = req.Username
	cfg.AuthMethod = authMethod
	if req.Password != "" {
		cfg.Password = req.Password // SECURITY WARNING: stored in plain text
	}
	if req.PrivateKey != "" {
		cfg.PrivateKey = req.PrivateKey
		cfg.Passphrase = req.Passphrase // update passphrase only when key changes
	}
	cfg.IsDefault = req.IsDefault

	if err := s.db.Save(&cfg).Error; err != nil {
		slog.Error("db.Save() error in UpdateSSHServerConfig", "config_id", cfg.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, toSSHResponse(cfg))
}

// DeleteSSHServerConfig deletes an SSH server config by ID.
func (s *Setting) DeleteSSHServerConfig(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		slog.Error("user not authenticated in DeleteSSHServerConfig")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("strconv.Atoi() error in DeleteSSHServerConfig", "id", c.Param("id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&table.SSHServerConfig{})
	if result.Error != nil {
		slog.Error("db.Delete() error in DeleteSSHServerConfig", "id", id, "user_id", userID, "err", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		slog.Error("SSH server config not found for deletion", "id", id, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
