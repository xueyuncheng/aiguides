package main

import (
	"aiguide/internal/app/aiguide"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateJWTSecret(t *testing.T) {
	secret, err := generateJWTSecret()
	if err != nil {
		t.Fatalf("generateJWTSecret() failed: %v", err)
	}

	if secret == "" {
		t.Error("generated JWT secret is empty")
	}

	if len(secret) < 32 {
		t.Errorf("generated JWT secret is too short: %d characters", len(secret))
	}

	// Generate another secret to ensure they are different
	secret2, err := generateJWTSecret()
	if err != nil {
		t.Fatalf("generateJWTSecret() failed: %v", err)
	}

	if secret == secret2 {
		t.Error("generated secrets should be different")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a test config
	testConfig := &aiguide.Config{
		APIKey:       "test_api_key",
		ModelName:    "test_model",
		UseGin:       true,
		GinPort:      "8080",
		JWTSecret:    "test_jwt_secret_12345678901234567890",
		AllowedEmails: []string{"test@example.com"},
	}

	// Save the config
	if err := saveConfig(tmpFile.Name(), testConfig); err != nil {
		t.Fatalf("saveConfig() failed: %v", err)
	}

	// Read the config back
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	loadedConfig := &aiguide.Config{}
	if err := yaml.Unmarshal(data, loadedConfig); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify the JWT secret was saved and loaded correctly
	if loadedConfig.JWTSecret != testConfig.JWTSecret {
		t.Errorf("JWT secret mismatch: expected %s, got %s", testConfig.JWTSecret, loadedConfig.JWTSecret)
	}
}

func TestAutoGenerateJWTSecret(t *testing.T) {
	// Create a temporary config file without JWT secret
	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a config without JWT secret
	initialConfig := &aiguide.Config{
		APIKey:    "test_api_key",
		ModelName: "test_model",
		UseGin:    true,
		GinPort:   "8080",
		AllowedEmails: []string{"test@example.com"},
	}

	data, err := yaml.Marshal(initialConfig)
	if err != nil {
		t.Fatalf("failed to marshal initial config: %v", err)
	}

	if err := os.WriteFile(tmpFile.Name(), data, 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	// Load the config (this should trigger JWT secret generation)
	data, err = os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	config := &aiguide.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify JWT secret is empty initially
	if config.JWTSecret != "" {
		t.Error("JWT secret should be empty initially")
	}

	// Simulate the auto-generation
	configModified := false
	if config.JWTSecret == "" {
		jwtSecret, err := generateJWTSecret()
		if err != nil {
			t.Fatalf("failed to generate JWT secret: %v", err)
		}
		config.JWTSecret = jwtSecret
		configModified = true
	}

	// Save the modified config
	if configModified {
		if err := saveConfig(tmpFile.Name(), config); err != nil {
			t.Fatalf("failed to save config file: %v", err)
		}
	}

	// Read the config again to verify it was saved
	data, err = os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	loadedConfig := &aiguide.Config{}
	if err := yaml.Unmarshal(data, loadedConfig); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify the JWT secret was generated and saved
	if loadedConfig.JWTSecret == "" {
		t.Error("JWT secret should have been generated and saved")
	}

	if loadedConfig.JWTSecret != config.JWTSecret {
		t.Error("JWT secret mismatch after reload")
	}
}
