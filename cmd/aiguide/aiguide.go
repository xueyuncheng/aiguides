package main

import (
	"aiguide/internal/app/aiguide"
	"context"
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
		slog.Error("failed to read config file", "err", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := &aiguide.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		slog.Error("failed to parse config yaml", "err", err)
		return fmt.Errorf("failed to parse config yaml: %w", err)
	}

	guide, err := aiguide.New(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	if err := guide.Run(ctx); err != nil {
		return fmt.Errorf("failed to run application: %w", err)
	}

	return nil
}
