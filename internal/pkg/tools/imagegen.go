package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"

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
	Success bool     `json:"success"`
	Images  []string `json:"images,omitempty"` // Base64编码的图片数据列表
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// NewImageGenTool 创建图片生成工具
//
// 该工具使用 Google Imagen 模型生成图片
// 如果传入 mockMode=true，将返回模拟的图片数据而不调用真实 API
func NewImageGenTool(client *genai.Client, mockMode bool) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "generate_image",
		Description: "生成 AI 图片。根据用户的文字描述生成相应的图片。支持指定图片数量（1-4张）、宽高比（1:1, 3:4, 4:3, 9:16, 16:9）等参数。",
	}

	handler := func(ctx tool.Context, input ImageGenInput) (*ImageGenOutput, error) {
		return generateImage(ctx, client, input, mockMode)
	}

	return functiontool.New(config, handler)
}

// generateImage 生成图片
func generateImage(ctx context.Context, client *genai.Client, input ImageGenInput, mockMode bool) (*ImageGenOutput, error) {
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

	slog.Info("开始生成图片", "prompt", input.Prompt, "number", numberOfImages, "aspect_ratio", aspectRatio, "mock_mode", mockMode)

	// 如果启用模拟模式，直接返回模拟数据
	if mockMode {
		return generateMockImages(numberOfImages, aspectRatio)
	}

	// 使用 GenerateContent API 生成图片
	model := DefaultImageModel

	// 构建提示词，包含数量和宽高比参数
	prompt := input.Prompt
	if input.NegativePrompt != "" {
		prompt = fmt.Sprintf("%s. Avoid: %s", prompt, input.NegativePrompt)
	}

	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		slog.Error("生成图片失败", "err", err)
		return &ImageGenOutput{
			Success: false,
			Error:   fmt.Sprintf("生成图片失败: %v", err),
		}, nil
	}

	if len(resp.Candidates) == 0 {
		slog.Error("没有生成任何图片")
		return &ImageGenOutput{
			Success: false,
			Error:   "没有生成任何图片",
		}, nil
	}

	// 转换图片为 Base64
	var images []string
	for _, part := range resp.Candidates[0].Content.Parts {
		// 处理文本部分
		if part.Text != "" {
			slog.Debug("模型响应文本", "text", part.Text)
			continue
		}

		// 处理内联图片数据
		if part.InlineData != nil {
			imageBytes := part.InlineData.Data
			if len(imageBytes) > 0 {
				base64Image := base64.StdEncoding.EncodeToString(imageBytes)
				// 添加 data URI 前缀
				mimeType := part.InlineData.MIMEType
				if mimeType == "" {
					mimeType = "image/png"
				}
				imageDataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
				images = append(images, imageDataURI)

				// 保存到本地看看
				if err := os.WriteFile("image.png", imageBytes, 0644); err != nil {
					slog.Error("保存图片到本地文件失败", "err", err)
					continue
				}
			}
		}
	}

	if len(images) == 0 {
		slog.Error("没有生成任何有效的图片")
		return &ImageGenOutput{
			Success: false,
			Error:   "没有生成任何有效的图片",
		}, nil
	}

	message := fmt.Sprintf("成功生成 %d 张图片", len(images))
	slog.Info(message)

	return &ImageGenOutput{
		Success: true,
		Images:  images,
		Message: message,
	}, nil
}

// generateMockImages 生成模拟的图片数据用于开发测试
// 避免每次都调用真实 API 花费费用
func generateMockImages(numberOfImages int32, aspectRatio string) (*ImageGenOutput, error) {
	var images []string

	// 生成指定数量的模拟图片
	for i := 0; i < int(numberOfImages); i++ {
		// 根据宽高比计算图片尺寸
		var width, height int
		switch aspectRatio {
		case "1:1":
			width, height = 256, 256
		case "3:4":
			width, height = 192, 256
		case "4:3":
			width, height = 256, 192
		case "9:16":
			width, height = 144, 256
		case "16:9":
			width, height = 256, 144
		default:
			width, height = 256, 256
		}

		// 创建一个简单的渐变图片
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		// 填充渐变颜色
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// 创建从蓝色到紫色的渐变效果
				r := uint8(50 + (x * 150 / width))
				g := uint8(150 - (x * 100 / width))
				b := uint8(200 - (y * 100 / height))
				img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}

		// 添加简单的边框
		for x := 0; x < width; x++ {
			img.SetRGBA(x, 0, color.RGBA{255, 255, 255, 255})
			img.SetRGBA(x, height-1, color.RGBA{255, 255, 255, 255})
		}
		for y := 0; y < height; y++ {
			img.SetRGBA(0, y, color.RGBA{255, 255, 255, 255})
			img.SetRGBA(width-1, y, color.RGBA{255, 255, 255, 255})
		}

		// 将图片转换为 base64
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			slog.Error("png.Encode() error", "err", err)
			continue
		}

		data, err := os.ReadFile("/Users/yuncheng/Documents/github/aiguides/cmd/aiguide/image1.png")
		if err != nil {
			slog.Error("os.ReadFile() error", "err", err)
			continue
		}

		// base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
		base64Image := base64.StdEncoding.EncodeToString(data)
		imageDataURI := fmt.Sprintf("data:image/png;base64,%s", base64Image)
		images = append(images, imageDataURI)
	}

	if len(images) == 0 {
		return &ImageGenOutput{
			Success: false,
			Error:   "模拟图片生成失败",
		}, nil
	}

	message := fmt.Sprintf("成功生成 %d 张模拟图片（开发模式）", len(images))
	slog.Info(message)

	return &ImageGenOutput{
		Success: true,
		Images:  images,
		Message: message,
	}, nil
}
