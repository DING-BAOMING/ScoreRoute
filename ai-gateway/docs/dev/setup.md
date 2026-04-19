# 开发环境搭建

## 环境要求

- Go 1.21+
- Node.js 18+
- SQLite

## 本地开发

### 1. 克隆代码
```bash
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute
```

### 2. 配置环境变量
```bash
cp .env.example .env
# 编辑.env设置密码和密钥
```

### 3. 启动服务
```bash
go run ./cmd/server
```

### 4. 前端开发
```bash
cd web
npm install
npm run dev
```

## Docker部署
```bash
docker-compose up -d
```
