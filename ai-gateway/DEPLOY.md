# ScoreRoute 部署指南

基于 Go + Vue3 + SQLite 的 AI API 网关系统。

## 快速部署 (Docker)

### 方式一：使用 Docker Compose（推荐）

```bash
# 克隆仓库
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute

# 创建环境配置文件
cat > .env << 'EOF'
PORT=3000
DATABASE_PATH=./data/gateway.db
LOG_PATH=./logs
ADMIN_PASSWORD=your_secure_password_here
JWT_SECRET=your_jwt_secret_here
EOF

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 方式二：手动构建

```bash
# 克隆仓库
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute

# 构建 Docker 镜像
docker build -t scoreroute .

# 运行容器
docker run -d \
  --name scoreroute \
  -p 3000:3000 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -e ADMIN_PASSWORD=your_password \
  -e JWT_SECRET=your_jwt_secret \
  scoreroute
```

## 手动部署

### 前置要求

- Go 1.21+
- Node.js 18+
- SQLite3

### 构建步骤

```bash
# 1. 克隆仓库
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute

# 2. 构建前端
cd web
npm install
npm run build
cd ..

# 3. 构建后端
go build -o server ./cmd/server

# 4. 创建必要的目录
mkdir -p data logs cache

# 5. 配置环境变量
cat > .env << 'EOF'
PORT=3000
DATABASE_PATH=./data/gateway.db
LOG_PATH=./logs
ADMIN_PASSWORD=your_secure_password_here
JWT_SECRET=your_jwt_secret_here
EOF

# 6. 启动服务
./server
```

## 配置说明

### 环境变量

| 变量 | 必填 | 说明 | 示例 |
|------|------|------|------|
| PORT | 否 | 服务端口，默认 3000 | 3000 |
| DATABASE_PATH | 否 | 数据库路径 | ./data/gateway.db |
| LOG_PATH | 否 | 日志目录 | ./logs |
| ADMIN_PASSWORD | **是** | 管理后台密码 | (设置强密码) |
| JWT_SECRET | **是** | JWT 签名密钥 | (设置随机字符串) |

### 获取 JWT_SECRET

```bash
# 使用 OpenSSL 生成随机字符串
openssl rand -base64 32
```

## 验证部署

```bash
# 检查服务是否启动
curl http://localhost:3000/health

# 访问管理后台
# 浏览器打开 http://localhost:3000
# 使用设置的 ADMIN_PASSWORD 登录
```

## Nginx 反向代理配置（可选）

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        
        # WebSocket 支持（如需要）
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 目录权限

```bash
# 确保目录存在并有适当权限
chmod 755 data logs cache
```

## 故障排查

### 服务启动失败

1. 检查日志: `docker-compose logs`
2. 确认环境变量已设置
3. 检查端口是否被占用: `lsof -i :3000`

### 数据库错误

1. 确认 data 目录存在
2. 检查目录权限
3. 确认 SQLite 库已安装

### 前端资源加载失败

1. 确认已执行 `npm run build`
2. 检查 `web/dist` 目录是否存在
3. 确认构建产物没有错误

## 更新升级

```bash
# 进入目录
cd ScoreRoute

# 拉取最新代码
git pull origin main

# 重新构建
docker-compose build

# 重启服务
docker-compose up -d
```

## 数据备份

```bash
# 备份数据库
cp data/gateway.db data/gateway.db.backup.$(date +%Y%m%d)

# 备份日志
tar -czf logs.backup.$(date +%Y%m%d).tar.gz logs/
```

## 更多信息

- 详细开发文档: [docs/开发计划.md](./docs/开发计划.md)
- API 文档: 登录后访问管理后台查看
