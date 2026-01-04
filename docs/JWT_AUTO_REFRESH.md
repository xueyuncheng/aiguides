# JWT Auto-Refresh Implementation

## 概述

本项目现已实现 JWT 自动刷新机制，使用访问令牌（Access Token）和刷新令牌（Refresh Token）的双令牌模式，提高了安全性和用户体验。

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
                    使用刷新令牌获取新的访问令牌
                            │
                            ▼
                    更新 auth_token Cookie
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
  "token_type": "Bearer",
  "expires_in": 900
}
```

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

## 前端集成建议

### 1. 自动刷新策略

前端可以实现以下策略来自动刷新令牌：

```typescript
// 检测 401 错误并自动刷新
async function authenticatedFetch(url: string, options?: RequestInit) {
  let response = await fetch(url, {
    ...options,
    credentials: 'include'
  });
  
  // 如果收到 401，尝试刷新令牌
  if (response.status === 401) {
    const refreshResponse = await fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include'
    });
    
    if (refreshResponse.ok) {
      // 刷新成功，重试原请求
      response = await fetch(url, {
        ...options,
        credentials: 'include'
      });
    } else {
      // 刷新失败，跳转到登录页
      window.location.href = '/login';
    }
  }
  
  return response;
}
```

### 2. 主动刷新策略

前端可以在访问令牌快过期时主动刷新：

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
4. **刷新令牌轮换**: 可以考虑在未来实现刷新令牌轮换机制（每次刷新时发放新的刷新令牌）

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

**重要**: 生产环境中请使用强随机密钥，并妥善保管。

## 迁移指南

现有系统迁移到新的刷新机制：

1. **无需代码更改**: 现有使用 `GenerateJWT()` 的代码无需修改
2. **建议升级**: 对于新的登录流程，建议使用 `GenerateTokenPair()` 生成双令牌
3. **前端更新**: 前端需要实现令牌刷新逻辑（参考上面的前端集成建议）

## 未来改进

1. **刷新令牌轮换**: 实现刷新令牌轮换机制，提高安全性
2. **令牌黑名单**: 实现令牌黑名单机制，支持强制登出
3. **设备管理**: 记录每个刷新令牌对应的设备，支持多设备管理
4. **滑动过期**: 实现刷新令牌的滑动过期机制，活跃用户自动延长有效期
