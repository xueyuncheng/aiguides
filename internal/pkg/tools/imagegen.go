package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// ValidAspectRatios 定义支持的宽高比
var ValidAspectRatios = map[string]bool{
	"1:1":  true,
	"3:4":  true,
	"4:3":  true,
	"9:16": true,
	"16:9": true,
}

// DefaultImageModel 定义默认使用的 Imagen 模型
const DefaultImageModel = "gemini-3-pro-image-preview"

// ImageGenInput 定义图片生成工具的输入参数
type ImageGenInput struct {
	Prompt         string `json:"prompt" jsonschema:"图片描述，详细描述要生成的图片内容"`
	NegativePrompt string `json:"negative_prompt,omitempty" jsonschema:"不想在图片中出现的内容描述（可选）"`
	NumberOfImages int32  `json:"number_of_images,omitempty" jsonschema:"要生成的图片数量，默认为1，最多4张"`
	AspectRatio    string `json:"aspect_ratio,omitempty" jsonschema:"图片宽高比，支持: 1:1, 3:4, 4:3, 9:16, 16:9，默认为1:1"`
}

// ImageGenOutput 定义图片生成工具的输出
type ImageGenOutput struct {
	Success        bool     `json:"success"`
	Images         []string `json:"images,omitempty"` // Base64编码的图片数据列表
	Message        string   `json:"message,omitempty"`
	Error          string   `json:"error,omitempty"`
	EnhancedPrompt string   `json:"enhanced_prompt,omitempty"` // 增强后的提示词
}

// NewImageGenTool 创建图片生成工具
//
// 该工具使用 Google Imagen 模型生成图片
func NewImageGenTool(client *genai.Client) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "generate_image",
		Description: "生成 AI 图片。根据用户的文字描述生成相应的图片。支持指定图片数量（1-4张）、宽高比（1:1, 3:4, 4:3, 9:16, 16:9）等参数。",
	}

	handler := func(ctx tool.Context, input ImageGenInput) (*ImageGenOutput, error) {
		return generateImage(ctx, client, input)
	}

	return functiontool.New(config, handler)
}

// generateImage 生成图片
func generateImage(ctx context.Context, client *genai.Client, input ImageGenInput) (*ImageGenOutput, error) {
	if input.Prompt == "" {
		slog.Error("图片描述不能为空")
		return &ImageGenOutput{
			Success: false,
			Error:   "图片描述不能为空",
		}, nil
	}

	// 设置默认值
	numberOfImages := input.NumberOfImages
	if numberOfImages <= 0 {
		numberOfImages = 1
	}
	if numberOfImages > 4 {
		numberOfImages = 4
	}

	aspectRatio := input.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}

	// 验证宽高比
	if !ValidAspectRatios[aspectRatio] {
		slog.Error("无效的宽高比", "aspect_ratio", aspectRatio)
		return &ImageGenOutput{
			Success: false,
			Error:   fmt.Sprintf("无效的宽高比: %s，支持的值: 1:1, 3:4, 4:3, 9:16, 16:9", aspectRatio),
		}, nil
	}

	// 配置图片生成参数
	config := &genai.GenerateImagesConfig{
		NumberOfImages: numberOfImages,
		AspectRatio:    aspectRatio,
		EnhancePrompt:  true, // 启用提示词增强
	}

	if input.NegativePrompt != "" {
		config.NegativePrompt = input.NegativePrompt
	}

	// 使用 Imagen 3 模型生成图片
	model := DefaultImageModel
	slog.Info("开始生成图片", "model", model, "prompt", input.Prompt, "number", numberOfImages, "aspect_ratio", aspectRatio)

	resp, err := client.Models.GenerateImages(ctx, model, input.Prompt, config)
	if err != nil {
		slog.Error("生成图片失败", "err", err)
		return &ImageGenOutput{
			Success: false,
			Error:   fmt.Sprintf("生成图片失败: %v", err),
		}, nil
	}

	if len(resp.GeneratedImages) == 0 {
		slog.Error("没有生成任何图片")
		return &ImageGenOutput{
			Success: false,
			Error:   "没有生成任何图片",
		}, nil
	}

	// 转换图片为 Base64
	var images []string
	var enhancedPrompt string
	for i, genImage := range resp.GeneratedImages {
		if genImage.Image == nil {
			slog.Warn("图片数据为空", "index", i)
			continue
		}

		// 获取增强后的提示词（只需要第一张图片的）
		if i == 0 && genImage.EnhancedPrompt != "" {
			enhancedPrompt = genImage.EnhancedPrompt
		}

		// 检查是否被过滤
		if genImage.RAIFilteredReason != "" {
			slog.Warn("图片被过滤", "reason", genImage.RAIFilteredReason)
			continue
		}

		// 将图片字节编码为 Base64
		if len(genImage.Image.ImageBytes) > 0 {
			base64Image := base64.StdEncoding.EncodeToString(genImage.Image.ImageBytes)
			// 添加 data URI 前缀
			mimeType := genImage.Image.MIMEType
			if mimeType == "" {
				mimeType = "image/png"
			}
			imageDataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
			images = append(images, imageDataURI)
		}
	}

	if len(images) == 0 {
		slog.Error("所有图片都被过滤或无效")
		return &ImageGenOutput{
			Success: false,
			Error:   "所有图片都被过滤或无效",
		}, nil
	}

	message := fmt.Sprintf("成功生成 %d 张图片", len(images))
	slog.Info(message)

	return &ImageGenOutput{
		Success:        true,
		Images:         images,
		Message:        message,
		EnhancedPrompt: enhancedPrompt,
	}, nil
}
