package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 生成访问令牌
	accessToken, err := authService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if accessToken == "" {
		t.Fatal("Expected non-empty access token")
	}

	// 验证访问令牌
	claims, err := authService.ValidateJWT(accessToken)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %s, got %s", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
	}

	if claims.Name != user.Name {
		t.Errorf("Expected Name %s, got %s", user.Name, claims.Name)
	}

	if claims.TokenType != "access" {
		t.Errorf("Expected TokenType 'access', got %s", claims.TokenType)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 生成刷新令牌
	refreshToken, err := authService.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if refreshToken == "" {
		t.Fatal("Expected non-empty refresh token")
	}

	// 验证刷新令牌
	claims, err := authService.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %s, got %s", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
	}

	if claims.Name != user.Name {
		t.Errorf("Expected Name %s, got %s", user.Name, claims.Name)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("Expected TokenType 'refresh', got %s", claims.TokenType)
	}
}

func TestGenerateTokenPair(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 生成令牌对
	tokenPair, err := authService.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Expected non-empty access token")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}

	if tokenPair.ExpiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("Expected ExpiresIn %d, got %d", int64((15 * time.Minute).Seconds()), tokenPair.ExpiresIn)
	}

	// 验证访问令牌
	accessClaims, err := authService.ValidateJWT(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if accessClaims.TokenType != "access" {
		t.Errorf("Expected access token type, got %s", accessClaims.TokenType)
	}

	// 验证刷新令牌
	refreshClaims, err := authService.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}

	if refreshClaims.TokenType != "refresh" {
		t.Errorf("Expected refresh token type, got %s", refreshClaims.TokenType)
	}
}

func TestValidateTokenWithWrongType(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 生成访问令牌
	accessToken, err := authService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// 尝试将访问令牌作为刷新令牌验证（应该失败）
	_, err = authService.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Error("Expected error when validating access token as refresh token")
	}

	// 生成刷新令牌
	refreshToken, err := authService.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	// 尝试将刷新令牌作为访问令牌验证（应该失败）
	_, err = authService.ValidateJWT(refreshToken)
	if err == nil {
		t.Error("Expected error when validating refresh token as access token")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	// 创建一个已过期的令牌
	expirationTime := time.Now().Add(-1 * time.Hour) // 1小时前过期
	claims := &Claims{
		UserID:    "test-user-id",
		Email:     "test@example.com",
		Name:      "Test User",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证过期令牌（应该失败）
	_, err = authService.ValidateJWT(expiredToken)
	if err == nil {
		t.Error("Expected error when validating expired token")
	}
}

func TestValidateInvalidSignature(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	// 使用不同的密钥创建令牌
	differentConfig := &Config{
		JWTSecret: "different-secret-key",
	}
	differentAuthService := NewAuthService(differentConfig)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 使用不同的密钥生成令牌
	token, err := differentAuthService.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// 尝试使用原始密钥验证（应该失败）
	_, err = authService.ValidateJWT(token)
	if err == nil {
		t.Error("Expected error when validating token with wrong signature")
	}
}

func TestGenerateStateToken(t *testing.T) {
	state1, err := GenerateStateToken()
	if err != nil {
		t.Fatalf("GenerateStateToken failed: %v", err)
	}

	if state1 == "" {
		t.Error("Expected non-empty state token")
	}

	// 生成第二个 state token，应该不同
	state2, err := GenerateStateToken()
	if err != nil {
		t.Fatalf("GenerateStateToken failed: %v", err)
	}

	if state1 == state2 {
		t.Error("Expected different state tokens on each call")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-testing",
	}
	authService := NewAuthService(config)

	user := &GoogleUser{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// 测试 GenerateJWT 仍然有效（向后兼容）
	token, err := authService.GenerateJWT(user)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	// 验证令牌是访问令牌
	claims, err := authService.ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.TokenType != "access" {
		t.Errorf("Expected GenerateJWT to generate access token, got %s", claims.TokenType)
	}
}
