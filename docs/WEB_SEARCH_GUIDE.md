# Web Search 功能使用指南

## 功能概述

AIGuides 现已集成 **Web Search** 功能，能够搜索互联网上的实时信息。该功能使用开源的 SearXNG 搜索引擎聚合服务，完全免费且无需 API Key。

## 主要特点

- ✅ **完全免费** - 无需任何 API Key
- ✅ **搜索全网** - 无域名限制
- ✅ **聚合多引擎** - 结果来自 Google、Bing、DuckDuckGo 等
- ✅ **自动故障转移** - 支持多个备用实例
- ✅ **保护隐私** - 不追踪用户

## 快速开始

### 1. 零配置使用（推荐）

如果不添加任何配置，系统会自动使用默认的 SearXNG 公共实例（`https://searx.be`），立即可用。

只需启动 AIGuides，Agent 就能自动使用 web_search 工具。

### 2. 自定义配置（可选）

如果想自定义 SearXNG 实例或添加备用实例，可以在 `cmd/aiguide/aiguide.local.yaml` 中添加：

```yaml
web_search:
  searxng:
    instance_url: "https://searx.be"  # 主实例
    fallback_instances:  # 备用实例（可选）
      - "https://search.sapti.me"
      - "https://searx.tiekoetter.com"
  default_language: "zh-CN"  # 默认搜索语言
  timeout_seconds: 30        # 请求超时时间
  max_results: 10            # 最大返回结果数
```

### 3. 启动服务

```bash
# 使用默认配置文件
./scripts/start.sh

# 或指定配置文件
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.local.yaml
```

## 使用示例

### 示例 1：搜索最新信息

**用户提问：**
> 2024年最流行的编程语言是什么？

**Agent 行为：**
1. 自动调用 `web_search` 工具
2. 搜索"2024年最流行的编程语言"
3. 综合多个搜索结果
4. 提供答案并附上信息来源链接

### 示例 2：实时数据查询

**用户提问：**
> Go 语言最新版本是多少？有哪些新特性？

**Agent 行为：**
1. 识别需要最新信息
2. 使用 web_search 搜索"Go 语言最新版本"
3. 提取关键信息
4. 总结新特性并提供官方链接

### 示例 3：技术问题查询

**用户提问：**
> React 19 正式发布了吗？

**Agent 行为：**
1. 搜索"React 19 发布"
2. 查看多个来源的信息
3. 给出确切答案和发布时间
4. 提供官方发布公告链接

### 示例 4：搜索后深入阅读

**用户提问：**
> 搜索 Go 1.23 的新特性，并详细告诉我 range over func 是怎么用的

**Agent 行为：**
1. 调用 `web_search` 搜索 "Go 1.23 新特性"
2. 获得官方博客链接
3. 调用 `web_fetch` 获取博客完整内容
4. 分析纯文本内容，找到 "range over func" 相关段落
5. 生成详细解答并附上具体代码示例

## 触发条件

Agent 会在以下情况自动使用 web_search 工具：

- 用户使用"搜索"、"查询"、"最新"、"现在"、"当前"等关键词
- 询问可能随时间变化的信息（如价格、新闻、天气等）
- Agent 不确定答案或信息可能过时时

## 搜索结果格式

Web Search 返回的结果包含：

```json
{
  "success": true,
  "results": [
    {
      "title": "结果标题",
      "link": "https://example.com",
      "snippet": "结果摘要...",
      "engine": "google"  // 来源引擎
    }
  ],
  "query": "搜索关键词",
  "message": "找到 5 个搜索结果"
}
```

## 可用的 SearXNG 公共实例

以下是一些稳定的 SearXNG 公共实例，可用作主实例或备用实例：

1. **https://searx.be** (推荐，默认)
2. **https://search.sapti.me**
3. **https://searx.tiekoetter.com**
4. **https://searx.work**
5. **https://searx.bar**

更多实例列表：https://searx.space/

## 故障转移机制

如果主实例不可用，系统会自动尝试备用实例：

```
尝试主实例 -> 失败 -> 尝试备用实例1 -> 失败 -> 尝试备用实例2 -> 成功
```

日志示例：
```
INFO  尝试 SearXNG 搜索 instance=https://searx.be attempt=1 total=3
WARN  SearXNG 实例失败，尝试下一个 instance=https://searx.be
INFO  尝试 SearXNG 搜索 instance=https://search.sapti.me attempt=2 total=3
INFO  搜索成功 instance=https://search.sapti.me results_count=5
```

## 性能优化建议

### 1. 配置多个备用实例
```yaml
web_search:
  searxng:
    instance_url: "https://searx.be"
    fallback_instances:
      - "https://search.sapti.me"
      - "https://searx.tiekoetter.com"
      - "https://searx.work"
```

### 2. 调整超时时间
```yaml
web_search:
  timeout_seconds: 20  # 根据网络情况调整
```

### 3. 限制结果数量
```yaml
web_search:
  max_results: 5  # 减少结果数量可以加快响应
```

## 自托管 SearXNG（高级）

如果想要更高的可用性和控制权，可以自己托管 SearXNG 实例：

### 使用 Docker 快速部署

```bash
# 拉取 SearXNG 镜像
docker pull searxng/searxng

# 运行 SearXNG
docker run -d \
  --name searxng \
  -p 8888:8080 \
  searxng/searxng

# 访问 http://localhost:8888
```

### 配置自托管实例

```yaml
web_search:
  searxng:
    instance_url: "http://localhost:8888"
```

## 故障排除

### 问题 1: 所有实例都不可用

**错误信息：**
```
ERROR 所有 SearXNG 实例都不可用
```

**解决方案：**
1. 检查网络连接
2. 尝试访问 https://searx.space/ 查找可用实例
3. 更新配置文件中的实例列表
4. 考虑自托管 SearXNG

### 问题 2: 请求超时

**错误信息：**
```
WARN SearXNG 实例失败 err="context deadline exceeded"
```

**解决方案：**
1. 增加超时时间：`timeout_seconds: 60`
2. 更换更快的实例
3. 检查代理设置（如果使用）

### 问题 3: 搜索结果为空

**返回信息：**
```json
{
  "success": false,
  "message": "未找到搜索结果"
}
```

**可能原因：**
- 搜索关键词过于生僻
- 实例过滤了某些内容
- 语言设置不匹配

**解决方案：**
- 更换搜索关键词
- 尝试不同的 SearXNG 实例
- 调整 `default_language` 设置

## 测试验证

### 运行单元测试

```bash
# 运行所有 web search 测试
go test ./internal/pkg/tools/... -v -run TestWebSearch

# 运行特定测试
go test ./internal/pkg/tools/... -v -run TestFallbackInstances
```

### 手动测试

启动 AIGuides 后，向 Agent 提问：

```
用户: 帮我搜索 Go 语言的最新资讯
Agent: [调用 web_search 工具] 根据搜索结果，Go 语言最近的资讯包括...
```

## 技术实现细节

### 文件结构

```
internal/pkg/tools/
├── websearch.go         # Web Search 工具实现
└── websearch_test.go    # 单元测试（15+ 测试用例）

internal/app/aiguide/
├── aiguide.go           # 配置解析和传递
└── assistant/
    ├── assistant.go     # Assistant 创建
    ├── agent.go         # Agent 注册工具
    └── assistant_agent_prompt.md  # 系统提示词
```

### API 调用流程

```
用户提问 
  -> Agent 识别需要搜索
  -> 调用 web_search 工具
  -> 构建 SearXNG API 请求
  -> 发送 HTTP GET 请求
  -> 解析 JSON 响应
  -> 返回搜索结果给 Agent
  -> Agent 综合结果回答用户
```

### 测试覆盖

- ✅ 工具创建和参数验证
- ✅ SearXNG API 调用（Mock Server）
- ✅ 空查询错误处理
- ✅ 故障转移机制
- ✅ 所有实例失败处理
- ✅ HTTP 超时处理
- ✅ 无效 JSON 响应处理
- ✅ HTTP 错误状态码处理
- ✅ 结果数量限制

## 配置参考

### 完整配置示例

```yaml
# AIGuides 配置文件
api_key: your_gemini_api_key
model_name: gemini-2.0-flash-exp

# Web Search 配置（完整示例）
web_search:
  searxng:
    instance_url: "https://searx.be"
    fallback_instances:
      - "https://search.sapti.me"
      - "https://searx.tiekoetter.com"
      - "https://searx.work"
  default_language: "zh-CN"
  timeout_seconds: 30
  max_results: 10
```

### 最小配置示例

```yaml
# 使用所有默认值
# web_search: {}
# 或者完全不配置，使用默认公共实例
```

## 常见问题 (FAQ)

### Q1: Web Search 需要 API Key 吗？
**A:** 不需要！完全免费，零配置即可使用。

### Q2: 有搜索次数限制吗？
**A:** 公共 SearXNG 实例可能有速率限制，但通常足够日常使用。如需更高频率，建议自托管。

### Q3: 支持哪些语言？
**A:** 支持所有主流语言，通过 `default_language` 配置，如 `zh-CN`（中文）、`en`（英文）等。

### Q4: 搜索结果的准确性如何？
**A:** SearXNG 聚合多个搜索引擎的结果（Google、Bing、DuckDuckGo 等），准确性很高。

### Q5: 能搜索图片吗？
**A:** 当前版本主要支持网页搜索。图片搜索功能可在未来版本添加。

### Q6: 可以限制搜索某些网站吗？
**A:** SearXNG 支持站点过滤，可以通过搜索语法实现，如 `site:go.dev golang`。

## Web Fetch Tool 使用指南

Web Fetch Tool 用于获取网页的完整内容和元数据，适合在 `web_search` 返回结果后进行深入阅读。

### 功能特点

- ✅ 自动抓取网页内容
- ✅ 提取纯文本正文（优先）
- ✅ 提取元数据（标题、作者、发布时间等）
- ✅ 自动过滤广告和无关内容

### 使用示例

#### 示例 1：直接抓取文章

**用户提问：**
> 帮我总结这篇文章 https://go.dev/blog/go1.23

**Agent 行为：**
1. 调用 `web_fetch` 获取网页完整内容
2. 使用纯文本正文进行分析
3. 输出文章摘要

#### 示例 2：搜索后抓取

**用户提问：**
> 搜索 Go 1.23 的新特性，并详细解释 range over func

**Agent 行为：**
1. 使用 `web_search` 搜索相关链接
2. 调用 `web_fetch` 抓取官方博客正文
3. 从纯文本中提取相关段落并解答

### 输入参数

```json
{
  "url": "https://example.com/article"
}
```

### 返回结果格式

```json
{
  "success": true,
  "url": "https://example.com/article",
  "title": "文章标题",
  "text_content": "正文纯文本内容...",
  "content": "<article>...</article>",
  "byline": "作者",
  "excerpt": "摘要...",
  "length": 1234,
  "site_name": "站点名称",
  "published_time": "2026-01-24T10:00:00Z",
  "modified_time": "2026-01-24T12:00:00Z",
  "image": "https://example.com/cover.jpg",
  "favicon": "https://example.com/favicon.ico",
  "language": "zh-CN",
  "message": "成功获取网页内容，共 1234 字"
}
```

### 使用建议

- 优先使用 `text_content`（纯文本）进行分析和总结
- 如果需要保留原文格式，可使用 `content`（HTML）
- 对于无法提取正文的页面，会返回错误信息

## 下一步计划

### 阶段二：Web Fetch Tool（✅ 已实现）
- ✅ 获取完整网页内容
- ✅ 提取正文（使用 go-readability v2）
- ✅ Agent 可以深入阅读搜索结果
- ✅ 提取元数据（标题、作者、发布时间等）
- ✅ 自动过滤广告和无关内容

详细说明请参考下方的 [Web Fetch Tool 使用指南](#web-fetch-tool-使用指南)。

### 阶段三：Real-time Data APIs（计划中）
- 天气 API
- 汇率 API
- 新闻 API
- 股票行情 API

## 贡献和反馈

如有问题或建议，欢迎：
- 提交 GitHub Issue
- 提交 Pull Request
- 参与项目讨论

## 许可证

MIT License
