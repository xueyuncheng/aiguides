package tools

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GoogleMapsInput 定义 Google Maps 工具的输入参数
type GoogleMapsInput struct {
	Locations []string `json:"locations" jsonschema:"要在地图上显示的位置列表，每个位置可以是地址、地点名称或坐标"`
}

// GoogleMapsOutput 定义 Google Maps 工具的输出
type GoogleMapsOutput struct {
	Success bool   `json:"success"`
	MapURL  string `json:"map_url,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewGoogleMapsTool 创建 Google Maps 地图生成工具。
//
// 该工具生成带有多个标记位置的 Google Maps URL，
// 用户可以在浏览器中打开该链接查看包含所有关键地点的地图。
func NewGoogleMapsTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "generate_google_maps",
		Description: "生成包含多个旅游景点的 Google Maps 地图链接，用于可视化旅游路线。输入景点位置列表（格式：'景点名, 城市'），输出可在浏览器打开的地图URL，显示所有景点和推荐路线。",
	}

	handler := func(ctx tool.Context, input GoogleMapsInput) (*GoogleMapsOutput, error) {
		return generateGoogleMapsURL(input), nil
	}

	return functiontool.New(config, handler)
}

// generateGoogleMapsURL 生成 Google Maps URL
func generateGoogleMapsURL(input GoogleMapsInput) *GoogleMapsOutput {
	if len(input.Locations) == 0 {
		slog.Error("位置列表不能为空")
		return &GoogleMapsOutput{
			Success: false,
			Error:   "位置列表不能为空",
		}
	}

	// 使用 Google Maps 的搜索功能，支持多个标记点
	// 格式: https://www.google.com/maps/dir/?api=1&destination=...&waypoints=...
	// 或使用搜索格式: https://www.google.com/maps/search/?api=1&query=...

	// 如果只有一个位置，使用简单的地图查询
	if len(input.Locations) == 1 {
		locationStr := strings.TrimSpace(input.Locations[0])
		if locationStr == "" {
			slog.Error("位置信息不能为空")
			return &GoogleMapsOutput{
				Success: false,
				Error:   "位置信息不能为空",
			}
		}
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(locationStr))

		return &GoogleMapsOutput{
			Success: true,
			MapURL:  mapURL,
			Message: "成功生成包含 1 个地点的地图链接",
		}
	}

	// 多个位置：构建包含所有位置的地图 URL
	// 使用 Google Maps 的 directions API 创建带多个途径点的路线
	// 格式: https://www.google.com/maps/dir/?api=1&origin=...&destination=...&waypoints=...

	var validLocations []string
	for i, loc := range input.Locations {
		locationStr := strings.TrimSpace(loc)
		if locationStr == "" {
			slog.Error("位置信息不能为空", "index", i)
			return &GoogleMapsOutput{
				Success: false,
				Error:   fmt.Sprintf("第 %d 个位置信息不能为空", i+1),
			}
		}
		validLocations = append(validLocations, locationStr)
	}

	// 使用第一个位置作为起点，最后一个位置作为终点，中间的作为途径点
	origin := url.QueryEscape(validLocations[0])
	destination := url.QueryEscape(validLocations[len(validLocations)-1])

	var mapURL string
	if len(validLocations) > 2 {
		// 有中间途径点
		waypoints := validLocations[1 : len(validLocations)-1]
		waypointsStr := strings.Join(waypoints, "|")
		mapURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&origin=%s&destination=%s&waypoints=%s&travelmode=walking",
			origin, destination, url.QueryEscape(waypointsStr))
	} else {
		// 只有起点和终点
		mapURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&origin=%s&destination=%s&travelmode=walking",
			origin, destination)
	}

	message := fmt.Sprintf("成功生成包含 %d 个地点的路线地图链接（按顺序连接）", len(input.Locations))

	return &GoogleMapsOutput{
		Success: true,
		MapURL:  mapURL,
		Message: message,
	}
}
