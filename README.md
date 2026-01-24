# aiguides

基于 Google ADK + Gemini 的全栈 AI 助手，支持多模态聊天、图片生成、网页搜索、邮件查询等功能。

## 主要功能

- 多模态对话（文字 + 图片输入）
- AI 图片生成（Imagen）
- 网页搜索与内容抓取
- 邮件查询（IMAP）
- 会话管理
- Google OAuth 登录

## 快速启动

1. **准备配置**
   ```bash
   cp cmd/aiguide/aiguide.yaml.example cmd/aiguide/aiguide.yaml
   ```
   编辑 `cmd/aiguide/aiguide.yaml`，填写 Gemini API Key：
   ```yaml
   api_key: "your_gemini_api_key_here"
   model_name: gemini-2.0-flash-exp
   ```

2. **启动服务**
   ```bash
   ./scripts/start.sh
   ```
   访问 http://localhost:3000

## 技术栈

- **后端**: Go 1.25+, Gin, GORM, SQLite, Google ADK
- **前端**: Next.js 15, React 19, TypeScript, Tailwind CSS
- **AI**: Google Gemini 2.0 + Imagen

## 手动启动

后端：
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
```

前端：
```bash
cd frontend && npm install && npm run dev
```

## Docker 部署

```bash
make build   # 构建镜像
make deploy  # 启动服务
```

## 许可证

MIT License
