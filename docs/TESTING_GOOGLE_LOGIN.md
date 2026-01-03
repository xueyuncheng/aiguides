# Google 登录功能测试指南

本文档提供了测试 AIGuide Google 登录功能的详细步骤。

## 测试前准备

### 1. 获取 Google OAuth 凭据

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用 Google+ API：
   - 导航到 "API 和服务" > "库"
   - 搜索 "Google+ API"
   - 点击启用

4. 创建 OAuth 2.0 凭据：
   - 导航到 "API 和服务" > "凭据"
   - 点击 "创建凭据" > "OAuth 客户端 ID"
   - 应用类型选择：**Web 应用**
   - 名称：AIGuide（或任意名称）
   - 授权的重定向 URI：添加以下 URL
     ```
     http://localhost:8080/auth/google/callback
     ```
   - 点击创建

5. 保存客户端 ID 和客户端密钥

### 2. 生成 JWT Secret

使用以下任一命令生成 JWT Secret：

```bash
# 使用 openssl
openssl rand -base64 32

# 使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"

# 使用 Node.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

### 3. 配置应用

编辑 `cmd/aiguide/aiguide.yaml`：

```yaml
api_key: YOUR_GEMINI_API_KEY
model_name: gemini-2.0-flash-exp
use_gin: true
gin_port: 8080

enable_authentication: true
google_client_id: YOUR_GOOGLE_CLIENT_ID
google_client_secret: YOUR_GOOGLE_CLIENT_SECRET
google_redirect_url: http://localhost:8080/auth/google/callback
jwt_secret: YOUR_GENERATED_JWT_SECRET
```

## 测试步骤

### 测试场景 1：未启用认证（默认模式）

1. **配置**：确保 `enable_authentication: false`

2. **启动服务**：
   ```bash
   ./scripts/start.sh
   ```

3. **验证**：
   - 访问 http://localhost:3000
   - 应该直接显示 Agent 选择页面，无需登录
   - 点击任意 Agent 卡片应该能直接进入对话页面

4. **预期结果**：
   - ✅ 无需登录即可使用所有功能
   - ✅ 页面右上角不显示用户信息

### 测试场景 2：启用认证

1. **配置**：设置 `enable_authentication: true` 并填写 OAuth 配置

2. **启动服务**：
   ```bash
   ./scripts/start.sh
   ```

3. **测试登录流程**：

   a. **访问首页**
   - 访问 http://localhost:3000
   - 应该自动重定向到登录页面（/login）
   
   b. **登录页面**
   - 显示 Google 登录按钮
   - 显示应用介绍和功能列表
   
   c. **点击登录**
   - 点击 "使用 Google 登录" 按钮
   - 重定向到 Google 授权页面
   
   d. **Google 授权**
   - 选择或输入 Google 账号
   - 授权应用访问基本信息
   
   e. **授权成功**
   - 自动重定向回应用首页
   - 右上角显示用户头像和名称
   - 可以正常使用所有 Agent 功能

4. **测试用户菜单**：
   - 点击右上角用户头像/名称
   - 应该显示下拉菜单
   - 包含用户信息和"退出登录"选项

5. **测试退出登录**：
   - 点击"退出登录"
   - 应该清除登录状态
   - 重定向到登录页面

6. **测试会话持久性**：
   - 登录成功后刷新页面
   - 应该保持登录状态（24小时内）
   - 无需重新登录

### 测试场景 3：API 认证

使用 curl 测试 API 认证：

1. **未认证时访问 API**：
   ```bash
   curl -X POST http://localhost:18080/api/assistant/chats/test-session \
     -H "Content-Type: application/json" \
     -d '{"user_id":"test","session_id":"test","message":"你好"}'
   ```
   预期结果：401 Unauthorized

2. **获取认证 token**（需要先在浏览器登录）：
   - 使用浏览器开发者工具
   - 查看 Cookie 中的 `auth_token`

3. **使用 token 访问 API**：
   ```bash
   curl -X POST http://localhost:18080/api/assistant/chats/test-session \
     -H "Content-Type: application/json" \
     -H "Cookie: auth_token=YOUR_TOKEN" \
     -d '{"user_id":"test","session_id":"test","message":"你好"}'
   ```
   预期结果：正常返回 SSE 流式响应

### 测试场景 4：健康检查

```bash
curl http://localhost:18080/health
```

预期结果：
```json
{"status":"ok"}
```

健康检查端点无需认证。

## 常见问题排查

### 1. 重定向错误

**问题**：OAuth 回调时显示 "redirect_uri_mismatch" 错误

**解决方案**：
- 检查 Google Cloud Console 中配置的重定向 URI
- 确保与配置文件中的 `google_redirect_url` 完全一致
- 确保包含 http:// 或 https:// 前缀

### 2. JWT 验证失败

**问题**：登录成功但随后请求返回 401

**解决方案**：
- 检查 `jwt_secret` 是否正确配置
- 确保前后端使用相同的 secret
- 检查浏览器 Cookie 是否被阻止

### 3. Cookie 未保存

**问题**：刷新页面后需要重新登录

**解决方案**：
- 检查浏览器 Cookie 设置
- 确保浏览器允许第三方 Cookie
- 检查 CORS 配置

### 4. 前端无法连接后端

**问题**：前端报告连接错误

**解决方案**：
- 确认后端已启动并监听 18080 端口
- 检查 `next.config.ts` 中的 rewrites 配置
- 检查防火墙设置

## 测试清单

完整测试清单：

- [ ] 场景 1：未启用认证，能正常使用
- [ ] 场景 2.a：启用认证后自动跳转登录页
- [ ] 场景 2.b：登录页正确显示
- [ ] 场景 2.c：点击登录跳转到 Google
- [ ] 场景 2.d：Google 授权流程正常
- [ ] 场景 2.e：授权后回到首页并显示用户信息
- [ ] 场景 2.4：用户菜单正确显示
- [ ] 场景 2.5：退出登录功能正常
- [ ] 场景 2.6：会话在刷新后保持
- [ ] 场景 3：API 认证正常工作
- [ ] 场景 4：健康检查正常
- [ ] 所有 Agent（assistant, web_summary, email_summary, travel）都能正常使用

## 安全注意事项

测试时需要注意的安全事项：

1. **不要在版本控制中提交敏感信息**
   - API Key
   - OAuth Client Secret
   - JWT Secret

2. **生产环境配置**
   - 使用 HTTPS
   - 更新重定向 URL
   - 使用环境变量存储敏感配置
   - 定期轮换 JWT Secret

3. **Token 安全**
   - JWT token 设置合理的过期时间（当前为 24 小时）
   - 使用 HTTP-only Cookie 防止 XSS 攻击
   - 实施 CSRF 保护

## 用户头像存储功能

### 功能说明

为了解决有时访问 Google 头像 URL 会返回错误的问题，系统现在会在用户登录时自动下载并存储用户的头像图片。

### 实现细节

1. **数据库存储**
   - `AvatarData`: 存储头像图片的二进制数据
   - `AvatarMimeType`: 存储图片的 MIME 类型（如 "image/jpeg", "image/png"）

2. **下载机制**
   - 首次登录时自动下载用户头像
   - 当头像 URL 变化时重新下载
   - 下载失败时优雅降级，继续使用原始 URL
   - 设置了超时限制（10秒）和大小限制（5MB）

3. **头像访问**
   - 新的 API 端点：`/api/auth/avatar/:userId`
   - 自动返回存储的头像图片
   - 如果没有存储的图片，则重定向到原始 Google URL
   - 设置了缓存头（Cache-Control: public, max-age=86400）以提高性能

### 测试头像功能

1. **使用 Google 登录**：
   ```bash
   # 登录后检查数据库
   sqlite3 aiguide.db "SELECT id, google_email, length(avatar_data), avatar_mime_type FROM users;"
   ```
   
   预期结果：应该看到 avatar_data 的长度（字节数）和 mime 类型

2. **访问头像 API**：
   ```bash
   # 获取用户 ID（从上一步或通过 /api/auth/user 获取）
   curl -I http://localhost:18080/api/auth/avatar/1
   ```
   
   预期结果：
   - 200 OK
   - Content-Type: image/jpeg 或 image/png
   - Cache-Control: public, max-age=86400

3. **检查前端显示**：
   - 登录后，用户头像应该显示在右上角
   - 检查浏览器开发者工具的 Network 标签
   - 头像请求应该指向 `/api/auth/avatar/:userId` 而不是外部 Google URL

4. **测试降级处理**：
   - 如果头像下载失败（例如网络问题），系统仍然会保存用户信息
   - 此时头像会显示 Google 的原始 URL（如果可用）
   - 下次登录时会尝试重新下载

### 头像功能的优势

- ✅ 提高可靠性：避免 Google URL 访问失败
- ✅ 减少外部依赖：头像存储在本地数据库
- ✅ 提升性能：带缓存头的本地图片加载更快
- ✅ 向后兼容：下载失败时自动降级到原始 URL
- ✅ 安全性：限制下载超时和文件大小，防止滥用

## 性能测试

可选的性能测试：

```bash
# 测试并发登录（需要先获取多个用户的 token）
ab -n 100 -c 10 -H "Cookie: auth_token=YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -p request.json \
  http://localhost:18080/api/assistant/chats/test
```

## 反馈

如果在测试过程中发现问题，请记录：
- 问题描述
- 复现步骤
- 预期行为
- 实际行为
- 浏览器/系统信息
- 错误日志
