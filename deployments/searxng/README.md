# SearXNG 本地搜索引擎部署指南

为 AIGuides 提供实时网络搜索能力的本地 SearXNG 搜索引擎部署配置。

## 📖 简介

SearXNG 是一个免费、开源的元搜索引擎，可以聚合来自多个搜索引擎的结果。本配置为 AIGuides 项目优化，提供稳定可靠的网络搜索能力。

**特性：**
- ✅ 聚合 Google、Bing、DuckDuckGo 三大主流搜索引擎
- ✅ 完全免费，无需 API 密钥
- ✅ 本地运行，无速率限制
- ✅ Redis 缓存，提升性能
- ✅ 中文搜索优化
- ✅ 隐私友好，不追踪用户

**系统要求：**
- Docker 20.10+
- Docker Compose V2
- 300-400 MB 可用内存
- 500 MB 可用磁盘空间

---

## 🚀 快速开始

### 前置要求

确保已安装 Docker：

```bash
docker --version
docker compose version
```

### 部署步骤

#### 1. 进入部署目录

```bash
cd deployments/searxng
```

#### 2. 启动服务

```bash
docker compose up -d
```

**预期输出：**
```
[+] Running 3/3
 ✔ Network aiguides-searxng-network  Created
 ✔ Container aiguides-redis          Started
 ✔ Container aiguides-searxng        Started
```

#### 3. 验证部署

检查容器状态：

```bash
docker compose ps
```

**预期输出：**
```
NAME                  IMAGE                      STATUS
aiguides-redis        redis:alpine               Up 10 seconds (healthy)
aiguides-searxng      searxng/searxng:latest     Up 10 seconds (healthy)
```

查看启动日志：

```bash
docker compose logs -f searxng
```

**成功标志：** 看到 `Application startup complete`

#### 4. 测试 Web 界面

浏览器打开：[http://localhost:8888](http://localhost:8888)

尝试搜索 "golang" 或其他关键词，验证是否返回结果。

#### 5. 测试 API

```bash
curl "http://localhost:8888/search?q=golang&format=json" | jq .
```

**预期输出：** 包含 `results` 数组的 JSON 数据，每个结果包含 `title`、`url`、`content` 字段。

---

## 🔗 与 AIGuides 集成

### 1. 配置 AIGuides

编辑 `cmd/aiguide/aiguide.local.yaml`（已自动添加），确认包含以下配置：

```yaml
# 实时信息查询配置
web_search:
  searxng:
    instance_url: "http://localhost:8888"
    fallback_instances: []  # 本地实例无需备用
  default_language: "zh-CN"
  timeout_seconds: 30
  max_results: 10
```

### 2. 重启 AIGuides

根据你的启动方式重启 AIGuides 服务以加载新配置。

### 3. 测试搜索功能

在 AIGuides 聊天界面中提问：

```
最新的 Go 1.23 有什么新特性？
```

或：

```
2026 年的人工智能发展趋势是什么？
```

**验证成功：**
- Agent 会自动调用 `web_search` 工具
- 返回的信息是最新的网络搜索结果
- 查看日志可以看到工具调用记录

---

## 🛠️ 管理命令

### 启动服务

```bash
cd deployments/searxng
docker compose up -d
```

### 停止服务

```bash
docker compose down
```

**注意：** 这会停止容器但保留数据（Redis 缓存、配置等）。

### 完全清理（删除所有数据）

```bash
docker compose down -v
rm -rf redis/ data/
```

**警告：** 这会删除所有缓存数据，需要重新初始化。

### 查看日志

```bash
# 查看所有日志
docker compose logs

# 实时跟踪日志
docker compose logs -f

# 只查看 SearXNG 日志
docker compose logs -f searxng

# 查看最近 100 行
docker compose logs --tail=100
```

### 重启服务

```bash
docker compose restart
```

### 更新镜像

```bash
# 拉取最新镜像
docker compose pull

# 重新启动（使用新镜像）
docker compose up -d
```

### 查看资源占用

```bash
docker stats aiguides-searxng aiguides-redis
```

### 进入容器调试

```bash
# 进入 SearXNG 容器
docker compose exec searxng sh

# 进入 Redis 容器
docker compose exec redis sh

# 查看 Redis 缓存统计
docker compose exec redis redis-cli INFO stats
```

### 清除 Redis 缓存

```bash
# 清空所有缓存
docker compose exec redis redis-cli FLUSHALL

# 查看缓存键数量
docker compose exec redis redis-cli DBSIZE
```

---

## ⚙️ 配置说明

### 修改端口

如果端口 8888 被占用，修改 `docker-compose.yaml`：

```yaml
services:
  searxng:
    ports:
      - "9999:8080"  # 改为 9999 或其他可用端口
```

同时更新 AIGuides 配置中的 `instance_url`。

### 调整搜索超时时间

编辑 `searxng/settings.yml`：

```yaml
outgoing:
  request_timeout: 5.0  # 从 10.0 改为 5.0（秒）
  max_request_timeout: 30.0
```

重启服务使配置生效：

```bash
docker compose restart
```

### 搜索语言设置

**注意**：SearXNG 配置中不需要设置 `default_lang`，因为：

1. **默认值 "auto"**：SearXNG 会自动检测请求的语言偏好
2. **AIGuides 动态指定**：AIGuides 在每次搜索时会通过 API 参数指定语言（默认 `zh-CN`）
3. **灵活性更好**：可以根据不同查询使用不同语言

如果需要修改 AIGuides 的默认搜索语言，编辑 `internal/app/aiguide/aiguide.go`：

```go
webSearchConfig := tools.WebSearchConfig{
    SearXNG: tools.SearXNGConfig{
        DefaultLanguage: "en",  // 改为英文
        // ...
    },
}
```

### 启用/禁用特定搜索引擎

编辑 `searxng/settings.yml`，找到对应引擎并修改 `disabled` 字段：

```yaml
engines:
  - name: google
    disabled: false  # 改为 true 禁用
```

### 添加更多搜索引擎

在 `searxng/settings.yaml` 的 `engines` 部分添加新引擎，例如添加百度：

```yaml
  - name: baidu
    engine: baidu
    shortcut: bd
    categories: [general]
    disabled: false
    timeout: 3.0
    weight: 1
```

完整的引擎列表参见：[SearXNG 支持的搜索引擎](https://docs.searxng.org/user/configured_engines.html)

### 配置代理（如需要）

如果需要为搜索引擎配置代理，编辑 `searxng/settings.yaml`：

```yaml
outgoing:
  proxies:
    http: http://host.docker.internal:7890
    https: http://host.docker.internal:7890
```

**注意：** Docker 容器访问宿主机使用 `host.docker.internal`，而不是 `localhost`。

### 调整 Redis 缓存策略

修改 `docker-compose.yaml` 中 Redis 的 `command` 参数：

```yaml
command: redis-server --save 300 10 --loglevel warning
# 说明：每 300 秒如果有至少 10 次写入，则保存数据
```

---

## 🐛 常见问题

### 1. 端口被占用

**错误信息：**
```
Error: bind: address already in use
```

**解决方案：**
- 方案 A：停止占用端口的服务
- 方案 B：修改 `docker-compose.yaml` 使用其他端口（见配置说明）

### 2. 搜索返回空结果

**可能原因：**
1. 搜索引擎连接失败（网络问题）
2. 搜索词过于生僻
3. 所有引擎都超时

**排查步骤：**

```bash
# 1. 查看日志检查错误
docker compose logs searxng | grep ERROR

# 2. 手动测试各个引擎
curl "http://localhost:8888/search?q=test&engines=google&format=json"
curl "http://localhost:8888/search?q=test&engines=bing&format=json"
curl "http://localhost:8888/search?q=test&engines=duckduckgo&format=json"

# 3. 检查网络连通性
docker compose exec searxng ping -c 3 www.google.com
```

### 3. Redis 连接失败

**错误信息：**
```
Error connecting to Redis
```

**解决方案：**

```bash
# 1. 检查 Redis 容器状态
docker compose ps redis

# 2. 重启 Redis
docker compose restart redis

# 3. 查看 Redis 日志
docker compose logs redis
```

### 4. 内存占用过高

**解决方案：**

方案 A：清理 Redis 缓存
```bash
docker compose exec redis redis-cli FLUSHALL
```

方案 B：限制 Docker 资源使用

在 `docker-compose.yaml` 中添加：

```yaml
services:
  searxng:
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: '0.5'
  redis:
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
```

### 5. 搜索速度慢

**优化建议：**

1. **增加超时时间**（`searxng/settings.yaml`）：
   ```yaml
   outgoing:
     request_timeout: 5.0
   ```

2. **减少搜索引擎数量**：只保留速度最快的引擎

3. **调整结果数量**（`aiguide.local.yaml`）：
   ```yaml
   web_search:
     max_results: 5  # 从 10 改为 5
   ```

4. **启用更激进的缓存**（`docker-compose.yaml`）：
   ```yaml
   command: redis-server --save 30 1 --loglevel warning
   ```

### 6. Docker 容器无法启动

**排查步骤：**

```bash
# 1. 查看详细日志
docker compose logs

# 2. 检查配置文件语法
docker compose config

# 3. 完全重新部署
docker compose down -v
docker compose up -d
```

---

## 📊 架构说明

### 组件关系

```
┌─────────────────────────────────────────────┐
│         AIGuides (Go Application)           │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │  web_search Tool                     │  │
│  │  (internal/pkg/tools/websearch.go)   │  │
│  └────────────────┬─────────────────────┘  │
└───────────────────┼─────────────────────────┘
                    │ HTTP (localhost:8888)
                    ▼
┌─────────────────────────────────────────────┐
│         Docker Network: searxng-network     │
│                                             │
│  ┌──────────────────┐   ┌───────────────┐  │
│  │   SearXNG        │───│     Redis     │  │
│  │  (Port 8888)     │   │   (Cache)     │  │
│  └────────┬─────────┘   └───────────────┘  │
└───────────┼─────────────────────────────────┘
            │
            ▼
  ┌─────────────────────┐
  │  Internet Search    │
  │  - Google           │
  │  - Bing             │
  │  - DuckDuckGo       │
  └─────────────────────┘
```

### 数据流

1. **用户查询** → AIGuides Agent 检测需要实时信息
2. **工具调用** → `web_search` 工具发送请求到 SearXNG
3. **搜索聚合** → SearXNG 并行查询 Google、Bing、DuckDuckGo
4. **缓存检查** → Redis 检查是否有缓存结果
5. **结果返回** → SearXNG 聚合并去重结果
6. **展示结果** → AIGuides 将结果总结后展示给用户

### 文件结构

```
deployments/searxng/
├── docker-compose.yaml         # Docker 编排配置
├── .gitignore                  # Git 忽略规则
├── README.md                   # 本文档
├── searxng/
│   ├── settings.yml            # SearXNG 主配置（注意：必须是 .yml 后缀）
│   └── limiter.toml            # 限流器配置
├── redis/                      # Redis 数据（gitignored）
│   └── dump.rdb                # Redis 持久化文件
└── data/                       # SearXNG 缓存（gitignored）
    └── faviconcache.db         # 网站图标缓存
```

**注意**：`settings.yml` 文件名由 SearXNG 硬编码要求，不能改为 `.yaml` 后缀。

---

## 💡 性能优化建议

### 1. 调整缓存策略

**针对开发环境**（频繁重启）：
```yaml
# docker-compose.yaml
command: redis-server --save 30 1
```

**针对生产环境**（稳定运行）：
```yaml
command: redis-server --save 300 10
```

### 2. 搜索引擎权重调整

编辑 `searxng/settings.yaml`，提高响应快的引擎权重：

```yaml
engines:
  - name: google
    weight: 2  # 权重更高，优先显示
  
  - name: bing
    weight: 1
  
  - name: duckduckgo
    weight: 0.5  # 权重较低
```

### 3. 禁用不需要的分类

如果只需要综合搜索，可以禁用其他分类以提升性能：

```yaml
engines:
  - name: google
    categories: [general]  # 只保留 general

  # 删除或注释掉 google images, google videos 等
```

### 4. 监控和告警

创建简单的健康检查脚本（`healthcheck.sh`）：

```bash
#!/bin/bash
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/healthz)
if [ "$response" != "200" ]; then
  echo "SearXNG is down! HTTP $response"
  exit 1
fi
echo "SearXNG is healthy"
```

---

## 📚 参考资源

### 官方文档
- [SearXNG 官方网站](https://docs.searxng.org/)
- [SearXNG GitHub](https://github.com/searxng/searxng)
- [支持的搜索引擎列表](https://docs.searxng.org/user/configured_engines.html)
- [配置说明文档](https://docs.searxng.org/admin/settings/)

### AIGuides 相关
- Web Search 实现代码：`internal/pkg/tools/websearch.go`
- Web Search 测试：`internal/pkg/tools/websearch_test.go`
- Web Search 使用指南：`docs/WEB_SEARCH_GUIDE.md`

### 社区资源
- [公共 SearXNG 实例列表](https://searx.space/)
- [SearXNG Docker 部署指南](https://github.com/searxng/searxng-docker)

---

## 🆘 获取帮助

如果遇到问题：

1. **查看日志**：`docker compose logs -f`
2. **查看本文档的常见问题部分**
3. **检查 SearXNG 官方文档**
4. **查看 AIGuides 项目的 Issue**

---

## 📝 更新日志

### v1.0.0 (2026-01-24)
- ✅ 初始版本
- ✅ 支持 Google、Bing、DuckDuckGo
- ✅ Redis 缓存支持
- ✅ 中文优化
- ✅ 完整的文档和故障排查指南

---

**部署愉快！** 🎉

如有问题，请参考上述常见问题部分或查看日志进行排查。
