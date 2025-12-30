package tools

import (
	"fmt"
	"net/url"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// Location 定义地图上的一个位置点
type Location struct {
	Name    string `json:"name" jsonschema:"位置名称"`
	Address string `json:"address" jsonschema:"位置地址或坐标"`
}

// GoogleMapsInput 定义 Google Maps 工具的输入参数
type GoogleMapsInput struct {
	Locations []Location `json:"locations" jsonschema:"要在地图上显示的位置列表"`
	MapTitle  string     `json:"map_title,omitempty" jsonschema:"地图标题（可选）"`
}

// GoogleMapsOutput 定义 Google Maps 工具的输出
type GoogleMapsOutput struct {
	Success  bool   `json:"success"`
	MapURL   string `json:"map_url,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// NewGoogleMapsTool 创建 Google Maps 地图生成工具。
//
// 该工具生成带有多个标记位置的 Google Maps URL，
// 用户可以在浏览器中打开该链接查看包含所有关键地点的地图。
func NewGoogleMapsTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "generate_google_maps",
		Description: "生成包含多个关键地点的 Google Maps 地图链接。接收位置列表，返回可在浏览器中打开的 Google Maps URL，地图上会标记所有指定的地点。",
	}

	handler := func(ctx tool.Context, input GoogleMapsInput) (*GoogleMapsOutput, error) {
		return generateGoogleMapsURL(input), nil
	}

	return functiontool.New(config, handler)
}

// generateGoogleMapsURL 生成 Google Maps URL
func generateGoogleMapsURL(input GoogleMapsInput) *GoogleMapsOutput {
	if len(input.Locations) == 0 {
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
		location := input.Locations[0]
		queryStr := buildLocationQuery(location)
		if queryStr == "" {
			return &GoogleMapsOutput{
				Success: false,
				Error:   "位置信息不完整：名称和地址不能都为空",
			}
		}
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(queryStr))
		
		return &GoogleMapsOutput{
			Success: true,
			MapURL:  mapURL,
			Message: fmt.Sprintf("成功生成包含 1 个地点的地图链接"),
		}
	}

	// 多个位置：构建包含所有位置的地图 URL
	// 使用 Google Maps 的 "Place IDs" 或 "query" 参数
	// 最佳方式：生成一个包含所有地点标记的 URL
	
	// 方法1：使用 directions API 创建带多个途径点的路线
	// 格式: https://www.google.com/maps/dir/?api=1&origin=...&destination=...&waypoints=...
	
	var queries []string
	for _, loc := range input.Locations {
		queryStr := buildLocationQuery(loc)
		if queryStr == "" {
			return &GoogleMapsOutput{
				Success: false,
				Error:   fmt.Sprintf("位置信息不完整：'%s' 的名称和地址不能都为空", loc.Name),
			}
		}
		queries = append(queries, queryStr)
	}

	// 使用第一个位置作为起点，最后一个位置作为终点，中间的作为途径点
	origin := url.QueryEscape(queries[0])
	destination := url.QueryEscape(queries[len(queries)-1])
	
	var mapURL string
	if len(queries) > 2 {
		// 有中间途径点
		waypoints := queries[1 : len(queries)-1]
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

// buildLocationQuery 构建位置查询字符串
func buildLocationQuery(loc Location) string {
	if loc.Address != "" {
		// 如果有地址，优先使用地址
		if loc.Name != "" {
			return fmt.Sprintf("%s, %s", loc.Name, loc.Address)
		}
		return loc.Address
	}
	// 只有名称
	if loc.Name != "" {
		return loc.Name
	}
	// 如果名称和地址都为空，返回空字符串（调用方需要处理）
	return ""
}
