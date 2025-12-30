# aiguides

一个基于 Google ADK（Agent Development Kit）构建的 AI 助手框架，提供多种智能代理服务。

## 功能特性

### 1. 信息检索和事实核查（AI Assistant）
一个双阶段的智能助手，通过搜索引擎获取信息并进行事实核查：
- **SearchAgent**: 专业的信息检索助手，使用 Google Search 获取准确全面的信息
- **FactCheckAgent**: 严谨的事实核查专家，验证信息准确性并提供可靠的最终答案

### 2. 网页内容总结（WebSummaryAgent）
专业的网页内容分析助手，能够：
- 访问指定 URL 的网页内容
- 提取和分析主要内容、结构和关键信息
- 生成清晰全面的内容总结报告

### 3. 邮件智能总结（EmailSummaryAgent）⭐ 新功能
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

## 快速开始

### 安装

```bash
go build -o aiguide cmd/aiguide/aiguide.go
```

### 配置

创建 `aiguide.yaml` 配置文件：

```yaml
api_key: your-google-ai-api-key
model_name: gemini-2.0-flash-exp  # 或其他支持的模型
proxy: ""  # 可选，设置代理
```

### 运行

```bash
./aiguide -f aiguide.yaml
```

应用将启动以下服务：
- Web UI: 可通过浏览器访问的用户界面
- REST API: 用于编程访问
- WebSocket: 实时通信接口

## 使用示例

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

### 网页总结

```
请帮我总结这个网页的内容：https://example.com
```

### 信息检索

```
什么是量子计算？它的主要应用有哪些？
```

## 技术架构

- **框架**: [Google ADK](https://github.com/google/adk) v0.3.0
- **AI 模型**: Google Gemini
- **语言**: Go 1.25.5
- **工具系统**: 
  - `functiontool`: 自定义工具开发
  - `geminitool`: Google 内置工具（如 Google Search）
- **代理类型**:
  - `llmagent`: 基于 LLM 的单一代理
  - `sequentialagent`: 顺序执行的工作流代理

## 项目结构

```
.
├── cmd/
│   └── aiguide/          # 应用入口
├── internal/
│   ├── app/aiguide/      # 代理定义
│   │   ├── aiguide.go    # 主应用逻辑
│   │   ├── introduce.go  # 信息检索和事实核查代理
│   │   ├── websummary.go # 网页总结代理
│   │   └── emailsummary.go # 邮件总结代理
│   └── pkg/tools/        # 自定义工具
│       ├── webfetch.go   # 网页获取工具
│       └── mailfetch.go  # Apple Mail 邮件获取工具
├── go.mod
└── README.md
```

## 开发

### 添加新的代理

1. 在 `internal/app/aiguide/` 中创建新的代理文件
2. 实现 `New*Agent(model model.LLM)` 函数
3. 在 `aiguide.go` 的 `New()` 函数中注册代理到 `MultiLoader`

### 添加新的工具

1. 在 `internal/pkg/tools/` 中创建工具文件
2. 定义输入/输出结构体
3. 使用 `functiontool.New()` 创建工具
4. 在相应的代理配置中添加工具

### 代码格式化

```bash
make fmt
```

### 测试

```bash
go test ./...
```

## 常见问题

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

## 许可证

[待添加]

## 贡献

欢迎提交 Issue 和 Pull Request！
