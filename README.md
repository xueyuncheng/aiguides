# aiguides

一个基于 Google Gemini 构建的多功能 AI 助手，支持项目信息检索、事实核查、图片生成和邮件查询。

## ✨ 主要功能

- **智能搜索**：内置 Google Search 工具，提供实时、准确的信息检索。
- **图片生成**：集成 Google Imagen 3，支持多种比例和风格的高质量图片生成。
- **多模态对话**：聊天支持文字与图片输入，方便直接上传图片提问。
- **邮件查询**：支持通过 IMAP 协议连接邮件服务器，查询和读取邮箱中的邮件。
- **会话管理**：提供完整的会话记录保存、切换与删除功能，支持 SQLite 持久化存储。
- **流式响应**：基于 Server-Sent Events (SSE) 的实时响应，提供流畅的打字机体验。
- **现代化 UI**：基于 Next.js 15 + React 19 构建的自适应界面，支持暗色模式。

## 🚀 快速启动

1. **克隆项目**
   ```bash
   git clone https://github.com/xueyuncheng/aiguides.git
   cd aiguides
   ```

2. **配置 API Key**
   编辑 `cmd/aiguide/aiguide.yaml` 文件，填入你的 Google Gemini API Key：
   ```yaml
   api_key: "your_api_key_here"
   ```

3. **一键启动**
   ```bash
   ./scripts/start.sh
   ```
   启动后即可访问 [http://localhost:3000](http://localhost:3000)。

## 🛠️ 技术栈

- **后端**: Go, Gin, [Google ADK](https://github.com/google/fun-with-goog-adk), SQLite
- **前端**: Next.js, React, TypeScript, Tailwind CSS
- **AI 模型**: Google Gemini 2.0, Imagen 3

## 📝 许可证

MIT License
