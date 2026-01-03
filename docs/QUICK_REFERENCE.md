# Google 登录功能快速参考

## 🚀 快速开始（5分钟设置）

### 1. 获取 Google OAuth 凭据（3分钟）

访问 https://console.cloud.google.com/

```
1. 创建/选择项目
2. 启用 Google+ API
3. 创建 OAuth 客户端 ID
   - 类型: Web 应用
   - 回调 URL: http://localhost:8080/auth/google/callback
4. 保存 Client ID 和 Client Secret
```

### 2. 生成 JWT Secret（10秒）

```bash
openssl rand -base64 32
```

### 3. 配置文件（1分钟）

编辑 `cmd/aiguide/aiguide.yaml`:

```yaml
api_key: your_gemini_api_key
model_name: gemini-2.0-flash-exp
use_gin: true
gin_port: 8080

enable_authentication: true
google_client_id: YOUR_CLIENT_ID
google_client_secret: YOUR_CLIENT_SECRET
google_redirect_url: http://localhost:8080/auth/google/callback
jwt_secret: YOUR_JWT_SECRET
```

### 4. 启动服务（1分钟）

```bash
./scripts/start.sh
```

打开 http://localhost:3000 - 完成！

## 📁 文件结构

```
Google 登录相关文件:
├── 后端
│   ├── internal/pkg/auth/
│   │   ├── auth.go           # OAuth & JWT 核心
│   │   └── middleware.go     # 认证中间件
│   └── internal/app/aiguide/agentmanager/
│       └── router.go         # 认证路由
│
├── 前端
│   ├── app/contexts/
│   │   └── AuthContext.tsx   # 认证状态
│   ├── app/login/
│   │   └── page.tsx          # 登录页面
│   └── app/page.tsx          # 首页（用户菜单）
│
└── 文档
    ├── README.md             # 主文档
    ├── TESTING_GOOGLE_LOGIN.md  # 测试指南
    └── IMPLEMENTATION_SUMMARY.md # 实现总结
```

## 🔑 API 端点

### 认证端点
```
GET  /auth/google/login      # 获取 OAuth URL
GET  /auth/google/callback   # OAuth 回调
POST /auth/logout            # 退出登录
GET  /auth/user              # 获取用户信息（需认证）
```

### 配置端点
```
GET  /config                 # 获取应用配置
GET  /health                 # 健康检查
```

### Agent API（条件认证）
```
POST /api/assistant/chats/:id
POST /api/web_summary/chats/:id
POST /api/email_summary/chats/:id
POST /api/travel/chats/:id
```

## 🎨 UI 组件

### 登录页面
- Google 登录按钮
- 应用介绍
- 功能列表
- 响应式设计

### 用户菜单
- 用户头像和名称
- 下拉菜单
- 退出登录按钮

## 🔒 安全检查清单

开发环境：
- [ ] JWT Secret 已生成（32+ 字符）
- [ ] Google OAuth 凭据已配置
- [ ] 回调 URL 正确设置
- [ ] CORS 限制到 localhost

生产环境（额外要求）：
- [ ] 使用 HTTPS
- [ ] 更新回调 URL 为生产域名
- [ ] CORS 限制到实际域名
- [ ] JWT Secret 使用环境变量
- [ ] 启用日志审计

## 📊 认证流程

```
用户访问 http://localhost:3000
        ↓
检查 /config (认证已启用?)
        ↓
    重定向到 /login
        ↓
点击 "使用 Google 登录"
        ↓
GET /auth/google/login
        ↓
重定向到 Google OAuth
        ↓
用户授权
        ↓
Google 回调 /auth/google/callback
        ↓
验证 + 生成 JWT
        ↓
设置 Cookie
        ↓
重定向到首页（已登录）
```

## 🛠️ 常用命令

```bash
# 构建后端
go build -o aiguide ./cmd/aiguide/

# 运行后端
./aiguide -f cmd/aiguide/aiguide.yaml

# 构建前端
cd frontend && npm run build

# 运行前端
cd frontend && npm run dev

# 一键启动（推荐）
./scripts/start.sh

# 生成 JWT Secret
openssl rand -base64 32

# 测试健康检查
curl http://localhost:18080/health

# 测试配置
curl http://localhost:18080/config

# 测试认证（需登录后获取 token）
curl -H "Cookie: auth_token=YOUR_TOKEN" \
  http://localhost:18080/auth/user
```

## ⚙️ 配置选项

| 配置项 | 必需 | 默认值 | 说明 |
|--------|------|--------|------|
| `enable_authentication` | 否 | false | 是否启用认证 |
| `google_client_id` | 是* | - | Google Client ID |
| `google_client_secret` | 是* | - | Google Client Secret |
| `google_redirect_url` | 是* | - | OAuth 回调 URL |
| `jwt_secret` | 是* | - | JWT 签名密钥 |
| `frontend_url` | 否 | http://localhost:3000 | 前端 URL |

*仅在 `enable_authentication=true` 时必需

## 🐛 常见问题

### 1. redirect_uri_mismatch
检查 Google Console 中的回调 URL 是否与配置一致

### 2. JWT 验证失败
确保前后端使用相同的 `jwt_secret`

### 3. Cookie 未保存
检查浏览器 Cookie 设置，确保允许第三方 Cookie

### 4. 认证后仍然 401
清除浏览器 Cookie 并重新登录

## 📝 测试清单

- [ ] 禁用认证模式正常工作
- [ ] 启用认证后跳转到登录页
- [ ] Google 登录流程正常
- [ ] 登录后显示用户信息
- [ ] 退出登录功能正常
- [ ] 刷新页面保持登录状态
- [ ] 所有 Agent 正常工作

## 📖 更多文档

- 详细配置: 查看 `README.md`
- 测试指南: 查看 `TESTING_GOOGLE_LOGIN.md`
- 技术细节: 查看 `IMPLEMENTATION_SUMMARY.md`

## 💡 提示

1. 开发时建议先禁用认证（`enable_authentication: false`）
2. 测试认证时建议使用 Chrome 开发者工具查看 Cookie
3. 生产环境必须使用 HTTPS
4. 定期更换 JWT Secret 提高安全性
5. 使用 `.gitignore` 保护 `aiguide.yaml` 文件

## 🤝 支持

如有问题，请参考：
1. README.md - 完整的设置指南
2. TESTING_GOOGLE_LOGIN.md - 详细的测试场景
3. IMPLEMENTATION_SUMMARY.md - 技术实现细节
4. GitHub Issues - 报告问题

---

**注意**: 本功能完全可选。如果不需要认证，只需保持 `enable_authentication: false` 即可。
