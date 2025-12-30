package aiguide

import (
	"aiguide/internal/pkg/tools"
	"fmt"
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
)

const travelAgentInstruction = `你是一个专业的旅游规划助手，负责为用户制定详细的旅游行程计划。

**核心职责：**
1. 根据用户提供的旅游时间（天数）和目的地（国家或城市）制定旅游计划
2. 使用 GoogleSearch 工具搜索目的地的热门景点、美食、文化、交通等信息
3. 综合考虑时间安排、景点分布、交通便利性等因素
4. 提供详细、可行的每日行程安排

**规划要求：**
1. **行程概览**
   - 总结目的地特色和旅游亮点
   - 说明最佳旅游季节和天气情况
   - 提供签证、货币、语言等基本信息

2. **每日详细行程**
   - 按天数划分，为每一天制定具体计划
   - 包括：上午、下午、晚上的活动安排
   - 推荐具体景点、餐厅、体验活动
   - 说明各景点之间的交通方式和预计时间
   - 估算每日大致费用

3. **实用建议**
   - 推荐住宿区域和酒店类型
   - 当地交通攻略（地铁、公交、打车等）
   - 必备物品清单
   - 安全注意事项
   - 当地礼仪和文化禁忌

4. **美食推荐**
   - 当地特色美食介绍
   - 推荐餐厅或美食街
   - 价格区间参考

5. **预算估算**
   - 交通费用（机票、当地交通）
   - 住宿费用
   - 餐饮费用
   - 景点门票
   - 购物和娱乐预算
   - 总预算范围

**输出格式：**
使用清晰的结构化格式：
- 使用标题和子标题组织内容
- 用列表展示关键信息
- 适当使用表格展示时间安排
- 提供相关参考链接

**可选功能 - 地图可视化：**
如果你认为有帮助，可以使用 generate_google_maps 工具为用户生成地图链接：
- 在行程规划完成后，如果有明确的地点信息（景点名称和城市），可以调用此工具
- 传入重要景点的名称和所在城市，工具会返回一个 Google Maps 链接
- 在输出末尾添加地图链接供用户参考

**注意事项：**
- 使用 GoogleSearch 工具搜索最新、准确的旅游信息
- 考虑季节性因素（淡旺季、节假日等）
- 行程安排要合理，避免过于紧凑或松散
- 提供多样化的选择，兼顾热门景点和小众体验
- 标注信息来源，提供可靠的参考链接`

func NewTravelAgent(model model.LLM) (agent.Agent, error) {
	// 创建 Google Maps 工具
	googleMapsTool, err := tools.NewGoogleMapsTool()
	if err != nil {
		return nil, fmt.Errorf("new google maps tool error, err = %w", err)
	}

	travelAgentConfig := llmagent.Config{
		Name:        "TravelAgent",
		Model:       model,
		Description: "专业的旅游规划助手，根据用户的旅游时间和目的地提供详细的旅游行程规划，并在地图上显示重点地点",
		Instruction: travelAgentInstruction,
		OutputKey:   "travel_agent_output",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
			googleMapsTool,
		},
	}
	agent, err := llmagent.New(travelAgentConfig)
	if err != nil {
		slog.Error("llmagent.New() error", "err", err)
		return nil, fmt.Errorf("llmagent.New() error, err = %w", err)
	}

	return agent, nil
}
