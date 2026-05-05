package aiguide

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/auth"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setOAuthStateCookie generates a CSRF state token and stores it in a cookie.
// Returns the state value for use in the OAuth URL, or an error.
func (a *AIGuide) setOAuthStateCookie(c *gin.Context) (string, error) {
	state, err := auth.GenerateStateToken()
	if err != nil {
		return "", err
	}
	secure := a.secureCookie()
	c.SetCookie("oauth_state", state, 600, "/", "", secure, true)
	return state, nil
}

// GoogleLogin 处理 Google 登录请求
func (a *AIGuide) GoogleLogin(c *gin.Context) {
	state, err := a.setOAuthStateCookie(c)
	if err != nil {
		slog.Error("failed to generate state token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	url := a.authService.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// GoogleReauth 触发强制重新授权，用于存量用户补充 Calendar scope。
func (a *AIGuide) GoogleReauth(c *gin.Context) {
	state, err := a.setOAuthStateCookie(c)
	if err != nil {
		slog.Error("failed to generate state token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	secure := a.secureCookie()
	// Mark this as a reauth so GoogleCallback can redirect back to the settings page.
	c.SetCookie("oauth_return_to", "/settings/google-calendar", 600, "/", "", secure, true)

	url := a.authService.GetAuthURLWithForceConsent(state)
	c.Redirect(http.StatusFound, url)
}

// GoogleCallback 处理 Google OAuth 回调
func (a *AIGuide) GoogleCallback(c *gin.Context) {
	// 验证 state
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		slog.Error("failed to get state cookie", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	state := c.Query("state")
	if state != stateCookie {
		slog.Error("state mismatch")
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}

	// 清除 state cookie
	secure := a.secureCookie()
	c.SetCookie("oauth_state", "", -1, "/", "", secure, true)

	// 获取授权码
	code := c.Query("code")
	if code == "" {
		slog.Error("no code in callback")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no code"})
		return
	}

	// 交换令牌
	token, err := a.authService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		slog.Error("failed to exchange token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token"})
		return
	}

	// 获取用户信息
	user, err := a.authService.GetGoogleUser(c.Request.Context(), token)
	if err != nil {
		slog.Error("failed to get user info", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	// 验证是否在允许登录的邮箱列表中
	frontendURL := a.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Determine redirect target: reauth flows return to the settings page.
	returnTo := frontendURL
	if returnToCookie, err := c.Cookie("oauth_return_to"); err == nil && returnToCookie != "" {
		returnTo = frontendURL + returnToCookie
		c.SetCookie("oauth_return_to", "", -1, "/", "", secure, true)
	}

	// 如果 allowed_emails 为空，则允许所有人登录
	allowed := len(a.config.AllowedEmails) == 0 || slices.Contains(a.config.AllowedEmails, user.Email)
	if !allowed {
		slog.Error("login attempt from unauthorized email", "email", user.Email)
		c.Redirect(http.StatusFound, frontendURL+"/login?error=unauthorized")
		return
	}

	// 保存用户信息到数据库
	dbUser, err := saveUser(a.db, user, token.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user info"})
		return
	}

	// 生成访问令牌和刷新令牌，使用内部用户 ID
	tokenPair, err := a.authService.GenerateTokenPair(dbUser.ID, user)
	if err != nil {
		slog.Error("failed to generate token pair", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// 设置访问令牌 cookie (15分钟)
	c.SetCookie("auth_token", tokenPair.AccessToken, 900, "/", "", secure, true)
	// 设置刷新令牌 cookie (7天)，路径限制为 /api/auth 以减少暴露
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 604800, "/api/auth", "", secure, true)

	// 重定向到前端
	c.Redirect(http.StatusFound, returnTo)
}

func saveUser(db *gorm.DB, user *auth.GoogleUser, refreshToken string) (*table.User, error) {
	var u table.User
	if err := db.Where("google_user_id = ?", user.ID).First(&u).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("db.First() error", "err", err)
			return nil, fmt.Errorf("db.First() error, err = %w", err)
		}

		// Download avatar image
		avatarData, mimeType, err := downloadAvatar(user.Picture)
		if err != nil {
			slog.Warn("failed to download avatar", "err", err, "url", user.Picture)
			// Continue without avatar data - store the URL anyway
		}

		u = table.User{
			GoogleUserID:            user.ID,
			GoogleEmail:             user.Email,
			GoogleName:              user.Name,
			Picture:                 user.Picture,
			AvatarData:              avatarData,
			AvatarMimeType:          mimeType,
			GoogleOAuthRefreshToken: refreshToken,
		}

		if err := db.Create(&u).Error; err != nil {
			slog.Error("db.Create() error", "err", err)
			return nil, fmt.Errorf("db.Create() error, err = %w", err)
		}
	} else {
		// Update existing user info with only the fields that may have changed.
		// Using Updates(map) instead of Save avoids overwriting all columns (including
		// large AvatarData) and prevents accidental zero-value overwrites.
		oldPictureURL := u.Picture
		updates := map[string]any{
			"google_email": user.Email,
			"google_name":  user.Name,
			"picture":      user.Picture,
		}

		if len(u.AvatarData) == 0 || oldPictureURL != user.Picture {
			avatarData, mimeType, err := downloadAvatar(user.Picture)
			if err != nil {
				slog.Warn("failed to download avatar", "err", err, "url", user.Picture)
			} else {
				updates["avatar_data"] = avatarData
				updates["avatar_mime_type"] = mimeType
			}
		}

		// Google only issues a refresh_token on first authorization or after revocation.
		// Only update when a new token is present to avoid clearing an existing one.
		if refreshToken != "" {
			updates["google_oauth_refresh_token"] = refreshToken
			slog.Info("saveUser: persisting new Google OAuth refresh token", "userID", u.ID)
		}

		if err := db.Model(&u).Updates(updates).Error; err != nil {
			slog.Error("db.Model().Updates() error", "err", err)
			return nil, fmt.Errorf("db.Model().Updates() error: %w", err)
		}
	}

	return &u, nil
}
