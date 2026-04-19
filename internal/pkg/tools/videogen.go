package tools

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/middleware"
	"aiguide/internal/pkg/storage"

	"aiguide/internal/app/aiguide/table"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

// DefaultVideoModel 定义默认使用的 Veo 模型
const DefaultVideoModel = "veo-3.1-generate-preview"

// validVideoAspectRatios 视频支持的宽高比
var validVideoAspectRatios = map[string]bool{
	"16:9": true,
	"9:16": true,
}

// validVideoResolutions 视频支持的分辨率
var validVideoResolutions = map[string]bool{
	"720p":  true,
	"1080p": true,
}

// validVideoDurations 视频支持的时长（秒）
var validVideoDurations = map[int32]bool{
	4: true,
	6: true,
	8: true,
}

// VideoGenInput 定义视频生成工具的输入参数
type VideoGenInput struct {
	Prompt          string `json:"prompt" jsonschema:"视频描述，详细描述要生成的视频内容，包括场景、动作、风格等"`
	NegativePrompt  string `json:"negative_prompt,omitempty" jsonschema:"不想在视频中出现的内容描述（可选）"`
	AspectRatio     string `json:"aspect_ratio,omitempty" jsonschema:"视频宽高比，支持: 16:9（横屏）, 9:16（竖屏），默认为16:9"`
	Resolution      string `json:"resolution,omitempty" jsonschema:"视频分辨率，支持: 720p, 1080p，默认为720p"`
	DurationSeconds int32  `json:"duration_seconds,omitempty" jsonschema:"视频时长（秒），支持: 4, 6, 8，默认为8"`
}

// VideoGenOutput 定义视频生成工具的输出
type VideoGenOutput struct {
	Success bool     `json:"success"`
	Videos  []string `json:"videos,omitempty"` // 视频文件的下载 URL 列表
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// NewVideoGenTool 创建视频生成工具
func NewVideoGenTool(client *genai.Client, db *gorm.DB, fileStore storage.FileStore, mockMode bool) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "generate_video",
		Description: "生成 AI 视频。根据用户的文字描述生成相应的视频。支持指定宽高比（16:9, 9:16）、分辨率（720p, 1080p）、时长（4/6/8秒）等参数。视频生成需要较长时间（约30秒到数分钟），请耐心等待。",
	}

	handler := func(ctx tool.Context, input VideoGenInput) (*VideoGenOutput, error) {
		return generateVideo(ctx, client, db, fileStore, input, mockMode)
	}

	return functiontool.New(config, handler)
}

// generateVideo 生成视频
func generateVideo(ctx context.Context, client *genai.Client, db *gorm.DB, fileStore storage.FileStore, input VideoGenInput, mockMode bool) (*VideoGenOutput, error) {
	if input.Prompt == "" {
		slog.Error("视频描述不能为空")
		return &VideoGenOutput{
			Success: false,
			Error:   "视频描述不能为空",
		}, nil
	}

	// 设置默认值
	aspectRatio := input.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	if !validVideoAspectRatios[aspectRatio] {
		return &VideoGenOutput{
			Success: false,
			Error:   fmt.Sprintf("无效的宽高比: %s，支持的值: 16:9, 9:16", aspectRatio),
		}, nil
	}

	resolution := input.Resolution
	if resolution == "" {
		resolution = "720p"
	}
	if !validVideoResolutions[resolution] {
		return &VideoGenOutput{
			Success: false,
			Error:   fmt.Sprintf("无效的分辨率: %s，支持的值: 720p, 1080p", resolution),
		}, nil
	}

	durationSeconds := input.DurationSeconds
	if durationSeconds == 0 {
		durationSeconds = 8
	}
	if !validVideoDurations[durationSeconds] {
		return &VideoGenOutput{
			Success: false,
			Error:   fmt.Sprintf("无效的时长: %d秒，支持的值: 4, 6, 8", durationSeconds),
		}, nil
	}

	slog.Info("开始生成视频", "prompt", input.Prompt, "aspect_ratio", aspectRatio, "resolution", resolution, "duration", durationSeconds, "mock_mode", mockMode)

	if mockMode {
		return generateMockVideo()
	}

	// 构建提示词
	prompt := input.Prompt
	if input.NegativePrompt != "" {
		prompt = fmt.Sprintf("%s. Avoid: %s", prompt, input.NegativePrompt)
	}

	// 调用 Veo API 生成视频（异步操作）
	genConfig := &genai.GenerateVideosConfig{
		AspectRatio:     aspectRatio,
		Resolution:      resolution,
		DurationSeconds: &durationSeconds,
	}

	operation, err := client.Models.GenerateVideos(ctx, DefaultVideoModel, prompt, nil, genConfig)
	if err != nil {
		slog.Error("client.Models.GenerateVideos() error", "err", err)
		return &VideoGenOutput{
			Success: false,
			Error:   fmt.Sprintf("视频生成请求失败: %v", err),
		}, nil
	}

	// 轮询等待视频生成完成
	pollInterval := 10 * time.Second
	maxWait := 10 * time.Minute
	startTime := time.Now()

	for !operation.Done {
		if time.Since(startTime) > maxWait {
			slog.Error("视频生成超时", "elapsed", time.Since(startTime))
			return &VideoGenOutput{
				Success: false,
				Error:   "视频生成超时，请稍后重试",
			}, nil
		}

		slog.Info("等待视频生成完成...", "elapsed", time.Since(startTime).Round(time.Second))
		time.Sleep(pollInterval)

		operation, err = client.Operations.GetVideosOperation(ctx, operation, nil)
		if err != nil {
			slog.Error("client.Operations.GetVideosOperation() error", "err", err)
			return &VideoGenOutput{
				Success: false,
				Error:   fmt.Sprintf("查询视频生成状态失败: %v", err),
			}, nil
		}
	}

	// 检查错误
	if operation.Error != nil {
		slog.Error("视频生成失败", "error", operation.Error)
		return &VideoGenOutput{
			Success: false,
			Error:   fmt.Sprintf("视频生成失败: %v", operation.Error),
		}, nil
	}

	if operation.Response == nil || len(operation.Response.GeneratedVideos) == 0 {
		slog.Error("没有生成任何视频")
		return &VideoGenOutput{
			Success: false,
			Error:   "没有生成任何视频",
		}, nil
	}

	// 下载并保存视频
	userID, _ := middleware.GetUserID(ctx)
	sessionID, _ := ctx.Value(constant.ContextKeySessionID).(string)

	var videoURLs []string
	for i, gv := range operation.Response.GeneratedVideos {
		data, err := client.Files.Download(ctx, genai.NewDownloadURIFromGeneratedVideo(gv), nil)
		if err != nil {
			slog.Error("下载视频失败", "index", i, "err", err)
			continue
		}

		fileName := fmt.Sprintf("generated_video_%d.mp4", i+1)
		mimeType := "video/mp4"
		if gv.Video != nil && gv.Video.MIMEType != "" {
			mimeType = gv.Video.MIMEType
		}

		// 保存到 FileStore
		meta, err := fileStore.Save(ctx, storage.SaveInput{
			UserID:    userID,
			SessionID: sessionID,
			FileName:  fileName,
			MimeType:  mimeType,
			Content:   bytes.NewReader(data),
			SizeBytes: int64(len(data)),
		})
		if err != nil {
			slog.Error("保存视频文件失败", "index", i, "err", err)
			continue
		}

		// 创建 FileAsset 记录
		asset := &table.FileAsset{
			UserID:       userID,
			SessionID:    sessionID,
			Kind:         constant.FileAssetKindGenerated,
			MimeType:     mimeType,
			OriginalName: fileName,
			StoragePath:  meta.StoragePath,
			SizeBytes:    meta.SizeBytes,
			SHA256:       meta.SHA256,
			Status:       constant.FileAssetStatusReady,
			TextStatus:   constant.PDFTextExtractStatusPending,
		}

		if err := db.Create(asset).Error; err != nil {
			slog.Error("db.Create() FileAsset error", "err", err, "file_name", fileName)
			continue
		}

		videoURL := fmt.Sprintf("/api/assistant/files/%d/download", asset.ID)
		videoURLs = append(videoURLs, videoURL)
	}

	if len(videoURLs) == 0 {
		return &VideoGenOutput{
			Success: false,
			Error:   "视频下载或保存失败",
		}, nil
	}

	message := fmt.Sprintf("成功生成 %d 个视频", len(videoURLs))
	slog.Info(message, "duration", time.Since(startTime).Round(time.Second))

	return &VideoGenOutput{
		Success: true,
		Videos:  videoURLs,
		Message: message,
	}, nil
}

// generateMockVideo 生成模拟视频数据用于开发测试
func generateMockVideo() (*VideoGenOutput, error) {
	slog.Info("生成模拟视频（开发模式）")

	return &VideoGenOutput{
		Success: true,
		Videos:  []string{"data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDEAAAAIZnJlZQAAA"},
		Message: "成功生成 1 个模拟视频（开发模式）",
	}, nil
}
