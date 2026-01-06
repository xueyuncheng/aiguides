package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestServerRestartPersistence tests that JWT secret persists across server restarts
func TestServerRestartPersistence(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Create initial config without JWT secret
	initialConfig := `api_key: test_api_key
model_name: gemini-2.0-flash-exp
use_gin: true
gin_port: 8080
allowed_emails:
  - test@example.com
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// First startup: Read config and generate JWT secret
	t.Log("Simulating first startup...")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	type Config struct {
		APIKey        string   `yaml:"api_key"`
		ModelName     string   `yaml:"model_name"`
		UseGin        bool     `yaml:"use_gin"`
		GinPort       string   `yaml:"gin_port"`
		JWTSecret     string   `yaml:"jwt_secret"`
		AllowedEmails []string `yaml:"allowed_emails"`
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.JWTSecret != "" {
		t.Fatal("JWT secret should be empty initially")
	}

	// Generate JWT secret (simulating first startup)
	if config.JWTSecret == "" {
		jwtSecret, err := generateJWTSecret()
		if err != nil {
			t.Fatalf("Failed to generate JWT secret: %v", err)
		}
		config.JWTSecret = jwtSecret
	}

	// Save config (simulating automatic save)
	savedData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, savedData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	firstJWTSecret := config.JWTSecret
	t.Logf("First startup: Generated JWT secret: %s", firstJWTSecret[:10]+"...")

	// Wait a bit to simulate time passing
	time.Sleep(100 * time.Millisecond)

	// Second startup: Read config again (simulating server restart)
	t.Log("Simulating server restart...")
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after restart: %v", err)
	}

	config2 := &Config{}
	if err := yaml.Unmarshal(data, config2); err != nil {
		t.Fatalf("Failed to unmarshal config after restart: %v", err)
	}

	// Verify JWT secret persists
	if config2.JWTSecret == "" {
		t.Fatal("JWT secret should not be empty after restart")
	}

	if config2.JWTSecret != firstJWTSecret {
		t.Errorf("JWT secret changed after restart: expected %s, got %s",
			firstJWTSecret[:10]+"...", config2.JWTSecret[:10]+"...")
	}

	t.Logf("Second startup: JWT secret persisted correctly: %s", config2.JWTSecret[:10]+"...")

	// Third startup: Simulate another restart
	t.Log("Simulating another server restart...")
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after second restart: %v", err)
	}

	config3 := &Config{}
	if err := yaml.Unmarshal(data, config3); err != nil {
		t.Fatalf("Failed to unmarshal config after second restart: %v", err)
	}

	// Verify JWT secret still persists
	if config3.JWTSecret != firstJWTSecret {
		t.Errorf("JWT secret changed after second restart: expected %s, got %s",
			firstJWTSecret[:10]+"...", config3.JWTSecret[:10]+"...")
	}

	t.Logf("Third startup: JWT secret still persisted: %s", config3.JWTSecret[:10]+"...")
	t.Log("✓ JWT secret persistence across restarts verified successfully")
}

// TestIntegrationWithApplication tests the full application flow
// Note: This test uses a simple process management approach with time.Sleep and Kill.
// In a production scenario, consider using more robust process management with context
// cancellation or proper shutdown signals.
func TestIntegrationWithApplication(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}

	// Create a temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Create initial config without JWT secret
	initialConfig := `api_key: test_api_key
model_name: gemini-2.0-flash-exp
use_gin: true
gin_port: 8081
allowed_emails:
  - test@example.com
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Build the application
	t.Log("Building application...")
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "aiguide"), "./")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build application: %v\n%s", err, output)
	}

	// Start the application (it should generate JWT secret and exit quickly since we don't have valid API key)
	t.Log("Starting application (first run)...")
	appPath := filepath.Join(tmpDir, "aiguide")
	startCmd := exec.Command(appPath, "-f", configPath)
	startCmd.Start()

	// Give it time to read config and generate JWT secret
	time.Sleep(2 * time.Second)

	// Kill the process
	if startCmd.Process != nil {
		startCmd.Process.Kill()
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	type Config struct {
		JWTSecret string `yaml:"jwt_secret"`
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.JWTSecret == "" {
		t.Fatal("JWT secret was not generated and saved by the application")
	}

	t.Logf("✓ Application successfully generated and saved JWT secret: %s", config.JWTSecret[:10]+"...")
}
