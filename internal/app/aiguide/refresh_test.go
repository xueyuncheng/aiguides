package aiguide

import (
	"aiguide/internal/pkg/auth"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRefreshTokenEndpoint(t *testing.T) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建认证服务
	authService := auth.NewAuthService(&auth.Config{
		JWTSecret: "test-secret-key-for-testing",
	})

	// 创建测试用户
	user := &auth.GoogleUser{
		ID:    "test-google-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}
	internalUserID := "1" // Simulated internal database ID

	// 生成令牌对
	tokenPair, err := authService.GenerateTokenPair(internalUserID, user)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 创建 AIGuide 实例（仅用于测试）
	secureCookieDefault := true
	aiGuide := &AIGuide{
		config: &Config{
			SecureCookie: &secureCookieDefault,
		},
		authService: authService,
	}

	// 创建测试路由
	router := gin.New()
	router.POST("/api/auth/refresh", aiGuide.RefreshToken)

	// 测试场景 1: 使用 Cookie 中的刷新令牌
	t.Run("refresh with cookie", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: tokenPair.RefreshToken,
		})

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["access_token"] == nil || response["access_token"] == "" {
			t.Error("Expected access_token in response")
		}

		// 验证返回新的刷新令牌（滑动过期）
		if response["refresh_token"] == nil || response["refresh_token"] == "" {
			t.Error("Expected refresh_token in response for sliding expiration")
		}

		if response["token_type"] != "Bearer" {
			t.Errorf("Expected token_type 'Bearer', got %v", response["token_type"])
		}

		if response["expires_in"] != float64(900) {
			t.Errorf("Expected expires_in 900, got %v", response["expires_in"])
		}

		// 验证 Cookie 中设置了新的刷新令牌
		cookies := w.Result().Cookies()
		var foundRefreshToken bool
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" && cookie.Value != "" {
				foundRefreshToken = true
				break
			}
		}
		if !foundRefreshToken {
			t.Error("Expected refresh_token cookie to be set")
		}
	})

	// 测试场景 2: 使用请求体中的刷新令牌
	t.Run("refresh with body", func(t *testing.T) {
		requestBody := map[string]string{
			"refresh_token": tokenPair.RefreshToken,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["access_token"] == nil || response["access_token"] == "" {
			t.Error("Expected access_token in response")
		}

		// 验证返回新的刷新令牌
		if response["refresh_token"] == nil || response["refresh_token"] == "" {
			t.Error("Expected refresh_token in response")
		}
	})

	// 测试场景 3: 使用无效的刷新令牌
	t.Run("refresh with invalid token", func(t *testing.T) {
		requestBody := map[string]string{
			"refresh_token": "invalid-token",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	// 测试场景 4: 使用访问令牌而非刷新令牌（应该失败）
	t.Run("refresh with access token", func(t *testing.T) {
		requestBody := map[string]string{
			"refresh_token": tokenPair.AccessToken, // 错误地使用了访问令牌
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	// 测试场景 5: 缺少刷新令牌
	t.Run("refresh without token", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/auth/refresh", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestLogoutHandlerClearsBothCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := auth.NewAuthService(&auth.Config{
		JWTSecret: "test-secret-key-for-testing",
	})

	secureCookieDefault := true
	aiGuide := &AIGuide{
		config: &Config{
			SecureCookie: &secureCookieDefault,
		},
		authService: authService,
	}

	router := gin.New()
	router.POST("/api/auth/logout", aiGuide.Logout)

	req, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 检查是否清除了两个 cookie
	cookies := w.Result().Cookies()
	if len(cookies) < 2 {
		t.Errorf("Expected at least 2 cookies to be set, got %d", len(cookies))
	}

	var authTokenCleared, refreshTokenCleared bool
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" && cookie.MaxAge == -1 {
			authTokenCleared = true
		}
		if cookie.Name == "refresh_token" && cookie.MaxAge == -1 {
			refreshTokenCleared = true
		}
	}

	if !authTokenCleared {
		t.Error("Expected auth_token cookie to be cleared")
	}

	if !refreshTokenCleared {
		t.Error("Expected refresh_token cookie to be cleared")
	}
}
