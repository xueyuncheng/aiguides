# AIGuide Frontend

这是 AIGuide 项目的 Next.js 前端界面。

## 功能特性

- 🎨 现代化的用户界面，支持暗色模式
- 🤖 与四个 AI 助手实时交互：
  - AI Assistant（信息检索和事实核查）
  - WebSummary Agent（网页内容分析）
  - EmailSummary Agent（邮件智能总结）
  - Travel Agent（旅游规划助手）
- 💬 流式响应的聊天界面
- 📱 响应式设计，支持移动端和桌面端
- 📝 **原始内容查看** - 查看和复制 AI 回答的原始 Markdown 内容 ⭐ 新功能
  - 悬停在 AI 回答上时显示操作按钮
  - 支持在渲染效果和原始 Markdown 之间切换
  - 一键复制原始 Markdown 内容到剪贴板

## 技术栈

- **框架**: Next.js 16 (App Router)
- **语言**: TypeScript
- **样式**: Tailwind CSS v4
- **UI**: React 19

## 快速开始

### 前置要求

- Node.js 20+ 
- npm 或 yarn

### 安装依赖

```bash
npm install
```

### 配置环境变量

复制 `.env.example` 到 `.env.local` 并配置后端 API 地址：

```bash
cp .env.example .env.local
```

默认后端地址为 `http://localhost:8080`。

### 启动开发服务器

```bash
npm run dev
```

前端将在 [http://localhost:3000](http://localhost:3000) 启动。

### 构建生产版本

```bash
npm run build
npm start
```

## 使用说明

### 启动完整服务

1. **启动后端服务**（在项目根目录）:
```bash
go run cmd/aiguide/aiguide.go -f cmd/aiguide/aiguide.yaml
```

后端将在 `http://localhost:8080` 启动。

2. **启动前端服务**（在 frontend 目录）:
```bash
cd frontend
npm run dev
```

前端将在 `http://localhost:3000` 启动。

3. **访问应用**:
打开浏览器访问 [http://localhost:3000](http://localhost:3000)

### 与 AI 助手交互

1. 在首页选择您想要使用的 AI 助手
2. 进入聊天界面后，可以：
   - 点击示例问题快速开始
   - 或在输入框中输入您自己的问题
3. AI 助手将实时流式返回回答
4. 查看和复制原始 Markdown 内容：
   - 将鼠标悬停在 AI 回答上，右上角会显示操作按钮
   - 点击「原始」按钮可以切换显示原始 Markdown 内容
   - 点击「复制」按钮可以一键复制原始内容到剪贴板
   - 点击「渲染」按钮可以切换回渲染效果

## 项目结构

```
frontend/
├── app/
│   ├── page.tsx              # 首页 - Agent 选择界面
│   ├── chat/
│   │   └── [agentId]/
│   │       └── page.tsx      # 聊天页面
│   ├── layout.tsx            # 根布局
│   └── globals.css           # 全局样式
├── public/                   # 静态资源
├── next.config.ts            # Next.js 配置
├── tailwind.config.ts        # Tailwind CSS 配置
└── package.json              # 依赖配置
```

## API 集成

前端通过 RESTful API 与后端通信：

- **Endpoint**: `POST /api/v1/agents/{agentId}/sessions/{sessionId}`
- **请求体**:
```json
{
  "message": "用户的问题"
}
```
- **响应**: Server-Sent Events (SSE) 格式的流式数据

## 开发

### 代码格式化

```bash
npm run lint
```

### 添加新功能

1. 在 `app/` 目录下创建新页面或组件
2. 使用 TypeScript 确保类型安全
3. 遵循现有的代码风格和结构

## 常见问题

### 无法连接到后端？

确保：
1. 后端服务正在运行（默认端口 8080）
2. `.env.local` 中的 `NEXT_PUBLIC_API_URL` 配置正确
3. 防火墙没有阻止端口访问

### 样式没有正确加载？

尝试：
```bash
npm run build
```
重新构建项目。

## 许可证

请参考项目根目录的 LICENSE 文件。

