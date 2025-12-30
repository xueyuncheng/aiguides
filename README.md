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

### 3. EmailSummaryAgent（邮件智能总结）⭐ 新功能
专为 macOS 用户设计的 Apple Mail 智能分析助手，能够：
- 从 Apple Mail 客户端读取邮件
- 智能识别重要和紧急邮件
- 按优先级（高/中/低）自动分类
- 生成结构化的邮件总结报告，包括：
  - 邮件总览统计
  - 重要邮件清单（主题、发件人、摘要）
  - 建议行动项

**使用要求：**
- macOS 系统
- Apple Mail 应用正在运行
- 已授予脚本访问权限

### 4. TravelAgent（旅游规划助手）⭐ 新增地图功能
专业的旅游规划助手，根据用户的旅游时间和目的地提供详细的旅游行程规划：
- 根据旅游天数和目的地制定详细行程
- 搜索热门景点、美食、文化、交通等信息
- 提供每日详细行程安排（上午、下午、晚上）
- 推荐住宿、交通、美食等实用信息
- 估算旅游预算
- **自动生成 Google Maps 地图链接，在地图上显示所有重点地点** 🗺️

## 快速开始

### 前置要求
- Go 1.25.5 或更高版本
- Google Gemini API Key
- Node.js 20+ 和 npm（用于前端界面）

### 一键启动 🚀（推荐）

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

3. 使用启动脚本同时启动后端和前端：
```bash
./start.sh
```

该脚本会自动：
- 安装前端依赖（如果还未安装）
- 启动后端服务（端口 8080）
- 启动前端服务（端口 3000）

4. 访问应用：
打开浏览器访问 [http://localhost:3000](http://localhost:3000)

### 手动启动

#### 后端安装与运行

1. 构建项目：
```bash
go build -o aiguide ./cmd/aiguide/
```

2. 运行后端服务：
```bash
./aiguide -f cmd/aiguide/aiguide.yaml
```

或直接运行：
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
```

后端服务将启动在 `http://localhost:8080`，提供 Web API 和内置的 Web UI。

#### 前端安装与运行 ⭐ 新增 Next.js 界面

本项目现在包含一个基于 Next.js 的现代化前端界面！

1. 进入前端目录：
```bash
cd frontend
```

2. 安装依赖：
```bash
npm install
```

3. 启动前端开发服务器：
```bash
npm run dev
```

前端将在 `http://localhost:3000` 启动。

4. 访问应用：
打开浏览器访问 [http://localhost:3000](http://localhost:3000)

**前端特性：**
- 🎨 现代化的用户界面，支持暗色模式
- 💬 实时流式对话交互
- 📱 响应式设计，完美支持移动端
- ⚡ 快速、流畅的用户体验

详细的前端使用说明请查看 [frontend/README.md](frontend/README.md)。

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
- **Google Maps 地图链接，可视化展示所有重点地点**

**地图功能特点：**
- 自动识别行程中的关键位置（景点、餐厅、酒店等）
- 生成可点击的 Google Maps 链接
- 支持单个地点或多个地点的路线规划
- 可以在浏览器中直接打开地图查看位置

### 邮件总结

在 Web UI 或 API 中与 EmailSummaryAgent 交互：

```
请帮我总结一下收件箱中的重要邮件
```

或指定参数：

```
请获取最近 20 封邮件并总结其中的重要内容
```

AI 助手将：
1. 调用 `fetch_apple_mail` 工具读取邮件
2. 分析邮件内容和重要性
3. 按优先级分类
4. 生成总结报告

## 项目结构

```
aiguides/
├── cmd/
│   └── aiguide/
│       ├── aiguide.go      # 应用入口
│       ├── aiguide.yaml    # 配置文件
│       └── logger.go       # 日志配置
├── frontend/               # Next.js 前端界面 ⭐ 新增
│   ├── app/
│   │   ├── page.tsx        # 首页 - Agent 选择界面
│   │   ├── chat/           # 聊天界面
│   │   ├── layout.tsx      # 根布局
│   │   └── globals.css     # 全局样式
│   ├── public/             # 静态资源
│   ├── next.config.ts      # Next.js 配置
│   └── package.json        # 前端依赖
├── internal/
│   ├── app/
│   │   └── aiguide/
│   │       ├── aiguide.go      # 主应用逻辑
│   │       ├── introduce.go    # 信息检索和事实核查代理
│   │       ├── websummary.go   # 网页总结代理
│   │       ├── emailsummary.go # 邮件总结代理
│   │       └── travelagent.go  # 旅游规划代理
│   └── pkg/
│       └── tools/
│           ├── webfetch.go     # 网页获取工具
│           ├── mailfetch.go    # Apple Mail 邮件获取工具
│           └── googlemaps.go   # Google Maps 地图生成工具
├── go.mod
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

要添加新的 AI 助手，请参考现有的 agent 实现（如 `travelagent.go` 或 `emailsummary.go`），遵循以下步骤：

1. 在 `internal/app/aiguide/` 目录下创建新的 agent 文件
2. 实现 `NewXXXAgent(model model.LLM)` 函数
3. 在 `aiguide.go` 的 `New()` 函数中注册新的 agent
4. 将新 agent 添加到 `agent.NewMultiLoader()` 中

### 添加新的工具

1. 在 `internal/pkg/tools/` 中创建工具文件
2. 定义输入/输出结构体
3. 使用 `functiontool.New()` 创建工具
4. 在相应的代理配置中添加工具

**已有工具示例：**
- `webfetch.go` - 网页内容获取工具
- `mailfetch.go` - Apple Mail 邮件读取工具
- `googlemaps.go` - Google Maps 地图链接生成工具

### 前端开发

1. 进入前端目录：`cd frontend`
2. 安装依赖：`npm install`
3. 启动开发服务器：`npm run dev`
4. 修改 `app/` 目录下的组件和页面
5. 使用 TypeScript 确保类型安全
6. 遵循现有的代码风格和 React 最佳实践

详细的前端开发说明请查看 [frontend/README.md](frontend/README.md)。

## 常见问题

### 前端无法连接到后端？

1. 确保后端服务正在运行（默认端口 8080）
2. 检查 `frontend/.env.local` 中的 `NEXT_PUBLIC_API_URL` 配置
3. 确认防火墙没有阻止端口访问

### EmailSummaryAgent 无法访问邮件？

1. 确保在 macOS 系统上运行
2. 确保 Apple Mail 应用正在运行
3. 首次使用时，系统可能会要求授予脚本访问权限
4. 在"系统偏好设置 > 安全性与隐私 > 隐私 > 自动化"中确认权限设置

### 如何自定义邮箱？

在与 EmailSummaryAgent 交互时可以指定：

```
请帮我总结"工作"邮箱中的邮件
```

默认使用 INBOX（收件箱）。

### Google Maps 地图如何使用？

TravelAgent 会自动在生成旅游行程时调用 Google Maps 工具：
1. 识别行程中的关键地点（景点、餐厅、酒店等）
2. 自动生成包含这些地点的 Google Maps 链接
3. 用户可以直接点击链接在浏览器中查看地图
4. 支持单个地点查看或多地点路线规划

无需任何配置，只需正常使用 TravelAgent 即可。

## 技术栈

- **框架**：Google ADK (Agent Development Kit)
- **模型**：Google Gemini
- **语言**：Go 1.25.5
- **工具**：GoogleSearch, 自定义工具（WebFetch、MailFetch、GoogleMaps）

## 许可证

请参考项目 LICENSE 文件。
