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

	guide, err := aiguide.New(ctx, config)
	if err != nil {
		return fmt.Errorf("aiguide.New() error, err = %w", err)
	}

	if err := guide.Start(ctx); err != nil {
		return fmt.Errorf("guide.Start() error, err = %w", err)
	}

	return nil
}
