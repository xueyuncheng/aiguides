# aiguides

一个基于 Google ADK (Agent Development Kit) 构建的 AI 助手框架，提供多种智能助手服务。

## 功能特性

本项目包含以下 AI 助手：

### 1. AI Assistant（信息检索和事实核查）
专门用于信息检索和事实核查的 AI 助手，包含两个子代理：
- **SearchAgent**：专业的信息检索助手，使用 GoogleSearch 工具获取准确、全面的信息
- **FactCheckAgent**：严谨的事实核查专家，验证信息准确性并提供可靠的最终答案

### 2. WebSummaryAgent（网页内容分析）
专业的网页内容分析助手，擅长访问网页并提供深度总结：
- 获取网页内容
- 分析主要内容和结构
- 提取关键要点和重要数据
- 生成结构化的总结报告

### 3. TravelAgent（旅游规划助手）
专业的旅游规划助手，根据用户的旅游时间和目的地提供详细的旅游行程规划：
- 根据旅游天数和目的地制定详细行程
- 搜索热门景点、美食、文化、交通等信息
- 提供每日详细行程安排（上午、下午、晚上）
- 推荐住宿、交通、美食等实用信息
- 估算旅游预算

## 快速开始

### 前置要求
- Go 1.25.5 或更高版本
- Google Gemini API Key

### 安装与运行

1. 克隆项目：
```bash
git clone https://github.com/xueyuncheng/aiguides.git
cd aiguides
```

2. 配置 API Key：
编辑 `cmd/aiguide/aiguide.yaml` 文件，设置你的 Google Gemini API Key：
```yaml
api_key: your_api_key_here
```

3. 构建项目：
```bash
go build -o aiguide ./cmd/aiguide/
```

4. 运行：
```bash
./aiguide -f cmd/aiguide/aiguide.yaml
```

或直接运行：
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
```

应用将启动 Web API 和 Web UI，您可以通过浏览器访问并与不同的 AI 助手进行交互。

## 使用示例

### 旅游规划助手使用示例

与 TravelAgent 交互时，您可以这样提问：

```
我计划去日本东京旅游 5 天，请帮我制定详细的旅游计划。
```

```
我想在泰国曼谷玩 3 天，预算有限，请推荐经济实惠的行程。
```

```
帮我规划一个巴黎 7 日游的行程，我对艺术和美食特别感兴趣。
```

AI 助手会为您提供：
- 目的地概览和基本信息
- 每日详细行程安排
- 景点推荐和交通指南
- 美食推荐和餐厅建议
- 住宿区域推荐
- 预算估算
- 实用旅游建议

## 项目结构

```
aiguides/
├── cmd/
│   └── aiguide/
│       ├── aiguide.go      # 应用入口
│       ├── aiguide.yaml    # 配置文件
│       └── logger.go       # 日志配置
├── internal/
│   ├── app/
│   │   └── aiguide/
│   │       ├── aiguide.go      # 主应用逻辑
│   │       ├── introduce.go    # 信息检索和事实核查代理
│   │       ├── websummary.go   # 网页总结代理
│   │       └── travelagent.go  # 旅游规划代理
│   └── pkg/
│       └── tools/
│           └── webfetch.go     # 网页获取工具
└── README.md
```

## 开发

### 代码格式化
```bash
make fmt
```

或：
```bash
go fmt ./...
```

### 添加新的 Agent

要添加新的 AI 助手，请参考现有的 agent 实现（如 `travelagent.go`），遵循以下步骤：

1. 在 `internal/app/aiguide/` 目录下创建新的 agent 文件
2. 实现 `NewXXXAgent(model model.LLM)` 函数
3. 在 `aiguide.go` 的 `New()` 函数中注册新的 agent
4. 将新 agent 添加到 `agent.NewMultiLoader()` 中

## 技术栈

- **框架**：Google ADK (Agent Development Kit)
- **模型**：Google Gemini
- **语言**：Go 1.25.5
- **工具**：GoogleSearch, 自定义工具（WebFetch）

## 许可证

请参考项目 LICENSE 文件。
