# aiguides

一个基于 Google ADK + Gemini 的全栈 AI 助手，支持多模态聊天、图片生成、邮件查询、会话管理与 OAuth 登录。

## ✨ 主要功能

- **多模态对话**：支持文字 + 图片输入，流式 SSE 输出。
- **图片生成**：集成 Imagen（`generate_image` 工具），支持多比例与多张输出。
- **邮件查询**：IMAP 邮件查询（`query_emails` 工具），带前端配置页面。
- **会话管理**：创建/切换/删除会话，支持历史分页与标题生成。
- **Google 登录**：OAuth + JWT Cookie，支持允许邮箱白名单。
- **现代化 UI**：Next.js App Router + Tailwind CSS。

## 🚀 快速启动

1. **克隆项目**
   ```bash
   git clone https://github.com/xueyuncheng/aiguides.git
   cd aiguides
   ```

2. **准备配置**
   复制示例配置：
   ```bash
   cp cmd/aiguide/aiguide.yaml.example cmd/aiguide/aiguide.yaml
   ```
   编辑 `cmd/aiguide/aiguide.yaml`，至少填写：
   ```yaml
   api_key: "your_gemini_api_key_here"
   model_name: gemini-2.0-flash-exp
   ```
   可选配置：OAuth、JWT、`frontend_url`、`allowed_emails`、`mock_image_generation` 等。

3. **一键启动（本地开发）**
   ```bash
   ./scripts/start.sh
   ```
   - 前端: http://localhost:3000
   - 后端: http://localhost:8080

## 🧭 项目结构概览

- `cmd/aiguide/`：应用入口与配置文件
- `internal/app/aiguide/`：后端核心（路由、迁移、OAuth、助手服务）
- `internal/app/aiguide/assistant/`：Agent、Runner、SSE、会话 API
- `internal/pkg/tools/`：图片生成与邮件查询工具
- `frontend/`：Next.js 前端（登录、聊天、邮箱配置）
- `deployments/`：Docker 相关配置

## 🔑 登录与邮件配置

- Google 登录：按 `cmd/aiguide/aiguide.yaml` 中的注释配置 OAuth。
- 邮件服务器配置入口：`/settings/email-server-configs`。

## 🛠️ 技术栈

- **后端**: Go, Gin, GORM, SQLite, Google ADK
- **前端**: Next.js 15, React 19, TypeScript, Tailwind CSS
- **AI 模型**: Google Gemini + Imagen

## 🧪 本地运行（手动）

- 后端：
  ```bash
  go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
  ```
- 前端：
  ```bash
  cd frontend
  npm install
  npm run dev
  ```

## 📝 许可证

MIT License
