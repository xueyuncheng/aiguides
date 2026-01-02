.PHONY: fmt build-backend build-frontend build load-images deploy down clean

fmt:
	go fmt ./...

# 构建后端 Docker 镜像
build-backend:
	docker build -f Dockerfile.backend -t aiguide-backend:latest .

# 构建前端 Docker 镜像
build-frontend:
	docker build -f Dockerfile.frontend -t aiguide-frontend:latest .

# 构建所有镜像
build: build-backend build-frontend

# 保存镜像到 tar 文件（用于传输到服务器）
save-images:
	docker save -o aiguide-backend.tar aiguide-backend:latest
	docker save -o aiguide-frontend.tar aiguide-frontend:latest

# 从 tar 文件加载镜像（在服务器上执行）
load-images:
	docker load -i aiguide-backend.tar
	docker load -i aiguide-frontend.tar

# 部署服务（停止旧服务，启动新服务）
deploy:
	docker compose down
	docker compose up -d

# 停止所有服务
down:
	docker compose down

# 清理镜像 tar 文件
clean:
	rm -f aiguide-backend.tar aiguide-frontend.tar

