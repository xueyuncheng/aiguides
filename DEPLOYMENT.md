# CI/CD 部署指南

本文档说明如何设置和使用自动化 CI/CD 流程来部署 AIGuide 项目。

## 概述

CI/CD 流程会在您向 `main` 分支推送代码时自动：
1. 构建前端和后端的 Docker 镜像
2. 将镜像传输到您的服务器
3. 在服务器上自动部署更新

## 必需的 GitHub Secrets

在 GitHub 仓库设置中，您需要配置以下 Secrets（Settings → Secrets and variables → Actions）：

| Secret 名称 | 说明 | 示例 |
|------------|------|------|
| `SSH_PRIVATE_KEY` | SSH 私钥（用于连接服务器） | 完整的私钥内容 |
| `SERVER_HOST` | 服务器地址 | `example.com` 或 `192.168.1.100` |
| `SERVER_USER` | SSH 用户名 | `ubuntu` 或 `root` |
| `DEPLOY_PATH` | 服务器上的部署目录 | `/home/ubuntu/aiguide` |

### 生成 SSH 密钥对

如果您还没有 SSH 密钥对，请在本地机器上运行：

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy"
```

然后将公钥添加到服务器的 `~/.ssh/authorized_keys` 文件中：

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@server
```

将私钥内容复制到 GitHub Secret `SSH_PRIVATE_KEY` 中：

```bash
cat ~/.ssh/id_ed25519
```

## 服务器准备

### 1. 安装必需的软件

在服务器上安装 Docker 和 Docker Compose：

```bash
# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Docker Compose
sudo apt-get update
sudo apt-get install docker-compose-plugin

# 将当前用户添加到 docker 组
sudo usermod -aG docker $USER
```

### 2. 安装 Make

```bash
sudo apt-get install make
```

### 3. 创建部署目录和配置文件

```bash
# 创建部署目录
mkdir -p /home/ubuntu/aiguide  # 或您在 DEPLOY_PATH 中指定的路径

# 创建配置目录
mkdir -p /home/ubuntu/aiguide/config
mkdir -p /home/ubuntu/aiguide/data

# 复制并编辑配置文件
# 您需要将 aiguide.yaml 配置文件复制到服务器的 config 目录
# 配置文件包含 API 密钥等敏感信息，不要提交到 Git
scp cmd/aiguide/aiguide.yaml your-server:/home/ubuntu/aiguide/config/

# 或者在服务器上直接创建和编辑配置文件
vim /home/ubuntu/aiguide/config/aiguide.yaml
```

**重要**: 配置文件 (`aiguide.yaml`) 包含 API 密钥和其他敏感信息，应该：
- 不要提交到 Git 仓库
- 直接在服务器上创建和管理
- 确保文件权限正确: `chmod 600 /home/ubuntu/aiguide/config/aiguide.yaml`

## 本地测试

在推送到 main 分支之前，您可以在本地测试 Docker 构建：

### 构建镜像

```bash
make build
```

### 本地运行

```bash
docker compose up -d
```

访问：
- 前端：http://localhost:3000
- 后端：http://localhost:8080

### 停止服务

```bash
make down
```

## CI/CD 工作流

当您向 `main` 分支推送代码时，GitHub Actions 会自动：

1. **构建阶段**
   - 使用 `Dockerfile.backend` 构建后端镜像
   - 使用 `Dockerfile.frontend` 构建前端镜像

2. **传输阶段**
   - 将镜像保存为 tar 文件
   - 通过 SCP 传输到服务器
   - 同时传输 `docker-compose.yml` 和 `Makefile`

3. **部署阶段**
   - 在服务器上加载镜像
   - 执行 `docker compose down` 停止旧服务
   - 执行 `docker compose up -d` 启动新服务
   - 清理临时文件

## Makefile 命令说明

| 命令 | 说明 |
|------|------|
| `make build` | 构建所有 Docker 镜像 |
| `make build-backend` | 只构建后端镜像 |
| `make build-frontend` | 只构建前端镜像 |
| `make save-images` | 保存镜像到 tar 文件 |
| `make load-images` | 从 tar 文件加载镜像 |
| `make deploy` | 部署服务（停止旧服务并启动新服务） |
| `make down` | 停止所有服务 |
| `make clean` | 清理镜像 tar 文件 |

## 故障排除

### 查看服务器日志

```bash
# 在服务器上
cd /home/ubuntu/aiguide  # 您的部署目录
docker compose logs -f
```

### 查看特定服务的日志

```bash
docker compose logs -f backend
docker compose logs -f frontend
```

### 手动重启服务

```bash
docker compose restart
```

### 检查服务状态

```bash
docker compose ps
```

## 配置文件说明

### `Dockerfile.backend`
- 使用多阶段构建减小镜像大小
- 基于 Alpine Linux 构建轻量级镜像
- 默认暴露 8080 端口
- **配置文件通过 volume 挂载**，不包含在镜像中以保护敏感信息

### `Dockerfile.frontend`
- 使用 Next.js standalone 输出模式
- 多阶段构建优化性能
- 默认暴露 3000 端口

### `docker-compose.yml`
- 定义前后端服务
- 配置网络连接
- 设置自动重启策略
- 后端配置文件挂载: `./config:/config:ro`（只读）
- 后端数据目录挂载: `./data:/app/data`（用于数据库等持久化数据）

### 目录结构

在服务器上，您的部署目录应该有以下结构：

```
/home/ubuntu/aiguide/
├── docker-compose.yml       # Docker 编排配置
├── Makefile                 # 部署命令
├── config/                  # 配置文件目录
│   └── aiguide.yaml        # 后端配置（包含 API 密钥）
├── data/                    # 数据持久化目录
│   └── aiguide.db          # SQLite 数据库（自动创建）
├── aiguide-backend.tar     # 后端镜像（临时）
└── aiguide-frontend.tar    # 前端镜像（临时）
```

## 安全建议

1. **保护 SSH 私钥**：确保 GitHub Secret 中的私钥安全
2. **限制 SSH 访问**：只允许必要的 IP 访问服务器
3. **使用防火墙**：配置服务器防火墙规则
4. **定期更新**：保持 Docker 和系统软件更新

## 环境变量配置

如果您需要配置环境变量（如 API 密钥、数据库连接等），请在 `docker-compose.yml` 中的 `environment` 部分添加，或创建 `.env` 文件。

注意：不要将敏感信息提交到 Git 仓库！

## 联系支持

如有问题，请创建 GitHub Issue。
