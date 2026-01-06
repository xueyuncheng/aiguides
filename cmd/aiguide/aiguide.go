package main

import (
	"aiguide/internal/app/aiguide"
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultGinPort 是 Gin 服务器的默认端口
	DefaultGinPort = 8080
)

func main() {
	initLogger()

	configFile := flag.String("f", "./aiguide.yaml", "配置文件路径")
	flag.Parse()

	if err := run(context.Background(), *configFile); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		slog.Error("os.ReadFile() error", "err", err)
		return fmt.Errorf("os.ReadFile() error, err = %w", err)
	}

	config := &aiguide.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		slog.Error("yaml.Unmarshal() error", "err", err)
		return fmt.Errorf("yaml.Unmarshal() error, err = %w", err)
	}

	// Automatically generate and persist JWT secret if not configured
	configModified := false
	if config.JWTSecret == "" {
		slog.Info("JWT secret not configured, generating a new one...")
		jwtSecret, err := generateJWTSecret()
		if err != nil {
			slog.Error("failed to generate JWT secret", "err", err)
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		config.JWTSecret = jwtSecret
		configModified = true
	}

	// Save the config file if it was modified
	if configModified {
		if err := saveConfig(file, config); err != nil {
			slog.Error("failed to save config file", "err", err)
			return fmt.Errorf("failed to save config file: %w", err)
		}
		slog.Info("JWT secret generated and saved to config file", "file", file)
	}

	guide, err := aiguide.New(ctx, config)
	if err != nil {
		return fmt.Errorf("aiguide.New() error, err = %w", err)
	}

	if err := guide.Run(ctx); err != nil {
		return fmt.Errorf("guide.Run() error, err = %w", err)
	}

	return nil
}

// generateJWTSecret generates a secure random JWT secret
func generateJWTSecret() (string, error) {
	// Generate 32 bytes of random data
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// saveConfig saves the configuration to a YAML file
func saveConfig(file string, config *aiguide.Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with appropriate permissions (0644 = rw-r--r--)
	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
