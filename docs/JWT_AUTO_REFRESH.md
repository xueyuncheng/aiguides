# JWT Auto-Refresh Implementation

## 概述

本项目现已实现 JWT 自动刷新机制，使用访问令牌（Access Token）和刷新令牌（Refresh Token）的双令牌模式，结合**滑动过期（Sliding Expiration）**机制，提高了安全性和用户体验。

## 核心特性

### 滑动过期机制 (Sliding Expiration)

- **问题**: 固定过期时间会导致活跃用户在 7 天后被强制重新登录，用户体验不佳
- **解决方案**: 每次使用刷新令牌时，系统会同时颁发新的访问令牌和新的刷新令牌
- **效果**: 只要用户保持活跃（至少每 7 天访问一次），就无需重新登录
- **安全性**: 非活跃用户（7 天未访问）需要重新认证，保证了安全性

## 架构设计

### 令牌类型

1. **访问令牌 (Access Token)**
   - 有效期：15 分钟
   - 用途：用于 API 请求的身份验证
   - 存储位置：Cookie (`auth_token`)
   - Token Type: `access`

2. **刷新令牌 (Refresh Token)**
   - 有效期：7 天
   - 用途：用于获取新的访问令牌
   - 存储位置：Cookie (`refresh_token`)
   - Token Type: `refresh`

### 工作流程

```
┌─────────────┐
│   用户登录   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│ Google OAuth 认证        │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ 生成访问令牌和刷新令牌   │
│ - Access Token (15分钟)  │
│ - Refresh Token (7天)   │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ 设置两个 Cookie          │
│ - auth_token            │
│ - refresh_token         │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ 用户访问受保护的 API     │
└──────┬──────────────────┘
       │
       ▼
    访问令牌有效？
       │
   是 ─┤
       │
   否 ─┼──────────────────────┐
       │                      │
       ▼                      ▼
 正常访问              调用刷新端点
                      /api/auth/refresh
                            │
                            ▼
                  验证刷新令牌并生成新令牌对
                  （滑动过期：同时更新刷新令牌）
                            │
                            ▼
                  更新两个 Cookie
                  - auth_token (新的访问令牌)
                  - refresh_token (新的刷新令牌)
                            │
                            ▼
                        继续访问
```

## API 端点

### 1. 刷新令牌端点

**端点**: `POST /api/auth/refresh`

**描述**: 使用刷新令牌获取新的访问令牌

**请求方式 1 - 使用 Cookie**:
```http
POST /api/auth/refresh
Cookie: refresh_token=<refresh_token>
```

**请求方式 2 - 使用请求体**:
```http
POST /api/auth/refresh
Content-Type: application/json

{
  "refresh_token": "<refresh_token>"
}
```

**成功响应** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**注意**: 响应中包含新的 `refresh_token`，实现了滑动过期机制。同时，两个令牌也会通过 Cookie 自动设置。

**错误响应**:
- `400 Bad Request`: 缺少刷新令牌
- `401 Unauthorized`: 刷新令牌无效或已过期

### 2. 登出端点

**端点**: `POST /api/auth/logout`

**描述**: 登出用户并清除访问令牌和刷新令牌

**响应**: 
- 清除 `auth_token` Cookie
- 清除 `refresh_token` Cookie

## 代码实现

### 核心组件

1. **Token 结构** (`internal/pkg/auth/auth.go`)
   ```go
   type Claims struct {
       UserID    string `json:"user_id"`
       Email     string `json:"email"`
       Name      string `json:"name"`
       TokenType string `json:"token_type"` // "access" or "refresh"
       jwt.RegisteredClaims
   }
   
   type TokenPair struct {
       AccessToken  string `json:"access_token"`
       RefreshToken string `json:"refresh_token"`
       ExpiresIn    int64  `json:"expires_in"`
   }
   ```

2. **令牌生成方法**
   - `GenerateTokenPair()`: 生成访问令牌和刷新令牌对
   - `GenerateAccessToken()`: 生成访问令牌（15分钟）
   - `GenerateRefreshToken()`: 生成刷新令牌（7天）

3. **令牌验证方法**
   - `ValidateJWT()`: 验证访问令牌
   - `ValidateRefreshToken()`: 验证刷新令牌
   - `ValidateToken()`: 通用令牌验证（带类型检查）

### 向后兼容性

原有的 `GenerateJWT()` 方法仍然保留，内部调用 `GenerateAccessToken()`，确保现有代码继续工作。

## 前端集成

### ✅ 已实现自动刷新

项目前端（`frontend/app/contexts/AuthContext.tsx`）已实现自动令牌刷新功能：

```typescript
const authenticatedFetch = async (input: RequestInfo | URL, init?: RequestInit) => {
  const response = await fetch(input, {
    ...init,
    credentials: 'include',
  });
  
  // 如果收到 401，尝试刷新令牌
  if (response.status === 401) {
    try {
      const refreshResponse = await fetch('/api/auth/refresh', {
        method: 'POST',
        credentials: 'include',
      });
      
      if (refreshResponse.ok) {
        // 令牌刷新成功，重试原请求
        const retryResponse = await fetch(input, {
          ...init,
          credentials: 'include',
        });
        return retryResponse;
      } else {
        // 刷新失败，跳转到登录页
        handleUnauthorized();
      }
    } catch (error) {
      console.error('Failed to refresh token:', error);
      handleUnauthorized();
    }
  }
  
  return response;
};
```

**使用方法：**

在任何需要认证的组件中使用 `useAuth` hook：

```typescript
import { useAuth } from '@/app/contexts/AuthContext';

function MyComponent() {
  const { authenticatedFetch } = useAuth();
  
  // 使用 authenticatedFetch 发送请求，自动处理令牌刷新
  const response = await authenticatedFetch('/api/some-endpoint', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}
```

### 其他可选策略

### 1. 主动刷新策略（可选）

前端也可以在访问令牌快过期时主动刷新：

```typescript
// 每 14 分钟刷新一次（访问令牌有效期 15 分钟）
setInterval(async () => {
  try {
    await fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include'
    });
  } catch (error) {
    console.error('Failed to refresh token:', error);
  }
}, 14 * 60 * 1000);
```

## 安全考虑

1. **令牌类型检查**: 访问令牌和刷新令牌不能互换使用，系统会验证令牌类型
2. **HttpOnly Cookie**: 所有令牌都存储在 HttpOnly Cookie 中，防止 XSS 攻击
3. **短期访问令牌**: 访问令牌仅 15 分钟有效，减少令牌泄露的风险
4. **滑动过期机制**: ✅ **已实现** - 每次刷新时同时发放新的刷新令牌，活跃用户无需重新登录
5. **非活跃用户保护**: 7 天未访问的用户需要重新认证，平衡了便利性和安全性
6. **Cookie 路径限制**: ✅ **已实现** - 刷新令牌 Cookie 路径设置为 `/api/auth`，仅在认证端点发送，大幅减少暴露风险
7. **Secure Cookie 标志**: ✅ **已实现** - Cookie 的 secure 标志默认为 `true`（生产环境），仅在本地开发时设置为 `false`

### Cookie Secure 标志配置

**默认行为（安全）**: Cookie 的 secure 标志默认为 `true`，适合生产环境（HTTPS）。

**本地开发配置**: 在 `aiguide.yaml` 中添加以下配置以允许 HTTP 访问：

```yaml
secure_cookie: false  # 仅用于本地开发，生产环境不要设置此项
```

**配置说明**:
- 不设置 `secure_cookie`：默认为 `true`（安全，生产环境）
- `secure_cookie: true`：显式启用安全模式（生产环境）
- `secure_cookie: false`：仅用于本地 HTTP 开发

**生产部署**: 生产环境应使用 HTTPS，并确保 `secure_cookie` 未设置或设置为 `true`。

## 测试

所有功能都有完整的单元测试和集成测试：

- `internal/pkg/auth/auth_test.go`: 令牌生成和验证的单元测试
- `internal/app/aiguide/refresh_test.go`: 刷新端点的集成测试

运行测试：
```bash
go test ./internal/pkg/auth/
go test ./internal/app/aiguide/
```

## 配置

JWT 密钥通过配置文件设置：

```yaml
jwt_secret: "your-secret-key-here"
```

**自动生成 JWT Secret（推荐）**：
- 如果配置文件中没有设置 `jwt_secret`，系统会在首次启动时自动生成一个安全的随机密钥
- 自动生成的密钥会保存到配置文件中，确保服务器重启后用户无需重新登录
- 密钥使用 32 字节随机数据，经过 Base64 URL 编码，安全性高

**手动配置（可选）**：
你也可以手动生成并配置 JWT secret：

```bash
# 使用 openssl 生成
openssl rand -base64 32

# 或使用 Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

**重要**: 
- 生产环境中请保护好 JWT secret，不要提交到版本控制系统
- 已生成的 JWT secret 不要随意修改，否则现有用户的登录令牌将失效

## 迁移指南

现有系统迁移到新的刷新机制：

1. **无需代码更改**: 现有使用 `GenerateJWT()` 的代码无需修改
2. **建议升级**: 对于新的登录流程，建议使用 `GenerateTokenPair()` 生成双令牌
3. **前端更新**: 前端需要实现令牌刷新逻辑（参考上面的前端集成建议）

## 未来改进

1. ~~**滑动过期机制**~~: ✅ **已实现** - 刷新时同时更新刷新令牌
2. **令牌黑名单**: 实现令牌黑名单机制，支持强制登出
3. **设备管理**: 记录每个刷新令牌对应的设备，支持多设备管理
4. **刷新令牌使用次数限制**: 限制单个刷新令牌的使用次数，防止滥用
