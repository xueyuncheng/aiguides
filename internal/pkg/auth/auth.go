package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleUser 表示从 Google 获取的用户信息
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// Claims 表示 JWT token 中的声明
type Claims struct {
	UserID       string `json:"user_id"`        // Internal database user ID
	GoogleUserID string `json:"google_user_id"` // Google's user ID
	Email        string `json:"email"`
	Name         string `json:"name"`
	TokenType    string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair 包含访问令牌和刷新令牌
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // 访问令牌过期时间（秒）
}

// Config OAuth 配置
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	JWTSecret    string
}

// AuthService 认证服务
type AuthService struct {
	config      *Config
	oauthConfig *oauth2.Config
}

// NewAuthService 创建新的认证服务
func NewAuthService(config *Config) *AuthService {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		config:      config,
		oauthConfig: oauthConfig,
	}
}

// GetAuthURL 获取 Google OAuth 认证 URL
func (s *AuthService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode 使用授权码交换访问令牌
func (s *AuthService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return s.oauthConfig.Exchange(ctx, code)
}

// GetGoogleUser 从 Google 获取用户信息
func (s *AuthService) GetGoogleUser(ctx context.Context, token *oauth2.Token) (*GoogleUser, error) {
	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Return a generic error to the caller to avoid leaking sensitive info
		// The response body is intentionally not included in the error
		return nil, fmt.Errorf("failed to get user info from Google API: status=%d", resp.StatusCode)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &user, nil
}

// GenerateJWT 生成 JWT token (保持向后兼容，默认生成访问令牌)
func (s *AuthService) GenerateJWT(internalUserID string, user *GoogleUser) (string, error) {
	return s.GenerateAccessToken(internalUserID, user)
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func (s *AuthService) GenerateTokenPair(internalUserID string, user *GoogleUser) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(internalUserID, user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(internalUserID, user)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64((15 * time.Minute).Seconds()), // 访问令牌 15 分钟有效
	}, nil
}

// GenerateAccessToken 生成访问令牌（短期有效）
func (s *AuthService) GenerateAccessToken(internalUserID string, user *GoogleUser) (string, error) {
	expirationTime := time.Now().Add(15 * time.Hour) // 访问令牌 15 分钟有效
	claims := &Claims{
		UserID:       internalUserID,
		GoogleUserID: user.ID,
		Email:        user.Email,
		Name:         user.Name,
		TokenType:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken 生成刷新令牌（长期有效）
func (s *AuthService) GenerateRefreshToken(internalUserID string, user *GoogleUser) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour) // 刷新令牌 7 天有效
	claims := &Claims{
		UserID:       internalUserID,
		GoogleUserID: user.ID,
		Email:        user.Email,
		Name:         user.Name,
		TokenType:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT 验证 JWT token（访问令牌）
func (s *AuthService) ValidateJWT(tokenString string) (*Claims, error) {
	return s.ValidateToken(tokenString, "access")
}

// ValidateRefreshToken 验证刷新令牌
func (s *AuthService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.ValidateToken(tokenString, "refresh")
}

// ValidateToken 验证 JWT token（通用方法）
func (s *AuthService) ValidateToken(tokenString string, expectedType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// 验证 token 类型
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.TokenType)
	}

	return claims, nil
}

// GenerateStateToken 生成随机 state token
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
