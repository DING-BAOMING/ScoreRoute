# 开发环境搭建

## 环境要求

- Go 1.21+
- Node.js 18+
- SQLite
- Git

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

### 3. 启动后端
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

## 生产环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| JWT_SECRET | JWT签名密钥 | 随机字符串 |
| ADMIN_PASSWORD | 管理员密码 | 强密码 |
| PORT | 监听端口 | 3000 |
| DATABASE_PATH | 数据库路径 | /app/data/gateway.db |

## 相关链接

- 官网: https://www.scoreroute.com
- 演示: https://demo.scoreroute.com
- API: https://api.scoreroute.com
