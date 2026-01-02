# CI/CD 实现总结

## 已完成的工作

本次实现为 AIGuide 项目创建了完整的 CI/CD 自动化部署流程。

### 1. Docker 配置文件

#### `Dockerfile.backend` - 后端镜像
- **多阶段构建**：使用 Go 1.25.5 Alpine 作为构建阶段，Alpine 3.19 作为运行阶段
- **最小化镜像**：只包含编译后的二进制文件和必要的 CA 证书
- **安全性**：配置文件通过 volume 挂载，不包含在镜像中
- **端口**：暴露 8080 端口

#### `Dockerfile.frontend` - 前端镜像
- **Next.js Standalone 模式**：启用 standalone 输出，减小镜像体积
- **多阶段构建**：依赖安装、构建、运行三个独立阶段
- **生产优化**：使用 Node.js 20 Alpine，创建非 root 用户运行
- **端口**：暴露 3000 端口

#### `docker-compose.yml` - 服务编排
- **网络隔离**：创建独立的 Docker 网络用于服务间通信
- **Volume 挂载**：
  - `./config:/config:ro` - 只读配置文件目录
  - `./data:/app/data` - 数据持久化目录
- **依赖管理**：前端服务依赖后端服务启动
- **自动重启**：配置 `restart: unless-stopped`

### 2. Makefile 命令

新增的 Make 命令用于简化 Docker 操作：

```makefile
make build              # 构建前后端镜像
make build-backend      # 只构建后端镜像
make build-frontend     # 只构建前端镜像
make save-images        # 保存镜像到 tar 文件
make load-images        # 从 tar 文件加载镜像
make deploy             # 部署服务（down + up）
make down               # 停止所有服务
make clean              # 清理 tar 文件
```

### 3. GitHub Actions 工作流

#### `.github/workflows/deploy.yml`

**触发条件**：向 `main` 分支推送代码

**工作流程**：
1. **构建阶段**
   - 检出代码
   - 设置 Docker Buildx
   - 构建后端和前端镜像
   - 保存镜像为 tar 文件

2. **传输阶段**
   - 设置 SSH 连接
   - 创建远程部署目录
   - 通过 SCP 传输镜像文件
   - 传输 docker-compose.yml 和 Makefile

3. **部署阶段**
   - SSH 到服务器
   - 加载 Docker 镜像
   - 执行 docker compose down
   - 执行 docker compose up -d
   - 清理临时文件

4. **清理阶段**
   - 删除本地 tar 文件

### 4. 配置和优化

#### `.dockerignore`
优化 Docker 构建上下文，排除不必要的文件：
- Git 相关文件
- 文档文件
- 编辑器配置
- Node modules 和构建产物
- 环境变量文件

#### `.gitignore`
新增排除规则：
- `*.tar` - Docker 镜像 tar 文件

#### `frontend/next.config.ts`
新增配置：
- `output: 'standalone'` - 启用 Next.js standalone 模式，用于 Docker 部署

### 5. 文档

#### `DEPLOYMENT.md`
详细的部署指南，包括：
- GitHub Secrets 配置说明
- SSH 密钥生成和配置
- 服务器准备步骤
- 本地测试方法
- CI/CD 工作流说明
- 故障排除指南
- 安全建议

## 使用流程

### 首次设置

1. **配置 GitHub Secrets**
   - `SSH_PRIVATE_KEY` - SSH 私钥
   - `SERVER_HOST` - 服务器地址
   - `SERVER_USER` - SSH 用户名
   - `DEPLOY_PATH` - 部署目录路径

2. **准备服务器**
   ```bash
   # 安装 Docker 和 Docker Compose
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   sudo apt-get install docker-compose-plugin make
   
   # 创建目录结构
   mkdir -p /home/ubuntu/aiguide/{config,data}
   
   # 上传配置文件
   scp cmd/aiguide/aiguide.yaml server:/home/ubuntu/aiguide/config/
   ```

3. **推送到 main 分支**
   - GitHub Actions 将自动构建和部署

### 日常使用

1. **开发和测试**
   - 在功能分支上开发
   - 本地测试：`make build && docker compose up`

2. **部署到生产**
   - 合并到 main 分支
   - GitHub Actions 自动执行部署

3. **监控和维护**
   - 查看日志：`docker compose logs -f`
   - 重启服务：`docker compose restart`
   - 手动部署：`make deploy`

## 技术特点

### 安全性
- ✅ 配置文件通过 volume 挂载，不存储在镜像中
- ✅ 使用 SSH 密钥进行服务器认证
- ✅ 敏感信息通过 GitHub Secrets 管理
- ✅ 配置文件以只读模式挂载

### 可维护性
- ✅ 多阶段构建减小镜像体积
- ✅ 清晰的 Makefile 命令
- ✅ 详细的文档和注释
- ✅ 标准化的目录结构

### 可靠性
- ✅ 自动重启策略
- ✅ 数据持久化
- ✅ 服务依赖管理
- ✅ 网络隔离

### 效率
- ✅ 自动化构建和部署
- ✅ 并行构建前后端镜像
- ✅ Docker 缓存优化
- ✅ 增量更新

## 架构图

```
┌─────────────────────────────────────────────────────────┐
│                    GitHub Repository                     │
│                                                          │
│  1. Push to main branch                                 │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              GitHub Actions Workflow                     │
│                                                          │
│  2. Build Docker Images                                 │
│     ├── Backend (Go)                                    │
│     └── Frontend (Next.js)                              │
│                                                          │
│  3. Save Images to TAR                                  │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼ SCP
┌─────────────────────────────────────────────────────────┐
│                   Remote Server                          │
│                                                          │
│  /home/ubuntu/aiguide/                                  │
│  ├── config/                                            │
│  │   └── aiguide.yaml (配置文件)                        │
│  ├── data/                                              │
│  │   └── aiguide.db (数据库)                           │
│  ├── docker-compose.yml                                 │
│  └── Makefile                                           │
│                                                          │
│  4. Load Images                                         │
│  5. Docker Compose Down                                 │
│  6. Docker Compose Up                                   │
│                                                          │
│  ┌─────────────────┐      ┌──────────────────┐         │
│  │   Frontend      │─────▶│    Backend       │         │
│  │   (Port 3000)   │      │   (Port 8080)    │         │
│  └─────────────────┘      └──────────────────┘         │
└─────────────────────────────────────────────────────────┘
                       │
                       ▼
              Users Access via Browser
          http://server:3000 (Frontend)
          http://server:8080 (Backend API)
```

## 下一步

1. **测试部署**
   - 配置 GitHub Secrets
   - 推送代码到 main 分支
   - 验证自动部署是否成功

2. **可选优化**
   - 添加健康检查
   - 配置 HTTPS（使用 Nginx 反向代理 + Let's Encrypt）
   - 添加监控和日志聚合
   - 配置备份策略

3. **生产环境考虑**
   - 使用环境变量管理不同环境的配置
   - 添加数据库备份脚本
   - 配置防火墙规则
   - 设置监控告警

## 问题排查

如遇到问题，请检查：
1. GitHub Secrets 是否正确配置
2. 服务器 SSH 访问是否正常
3. Docker 和 Docker Compose 是否已安装
4. 配置文件是否正确放置在 `config/` 目录
5. 查看 GitHub Actions 日志
6. 查看服务器上的 Docker 日志：`docker compose logs`

更多详细信息请参考 `DEPLOYMENT.md`。
