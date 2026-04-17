# ScoreRoute - API网关管理系统

基于Go + Vue3的AI API网关系统，支持多种AI模型的统一接入和轮询调度。

## 功能特性

- **API接入管理**: 支持接入多种格式的AI API（OpenAI、Anthropic等）
- **模型管理**: 在接入的API下添加和管理多个模型
- **接出管理**: 生成外部可调用的API Key
- **轮询调度**: 根据格式和类型自动轮询调用底层API
- **调用日志**: 记录每次调用的详细信息（延迟、Token消耗等）
- **单用户认证**: 简单的登录验证机制

## 快速开始

### Docker部署

```bash
# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 手动部署

```bash
# 构建前端
cd web && npm install && npm run build && cd ..

# 构建后端
go build -o server ./cmd/server

# 运行
./server
```

## 访问地址

- 管理后台: http://localhost:3000
- API端点: http://localhost:3000/v1

## 默认登录信息

- 用户名: admin
- 密码: (请在 .env 文件中设置 ADMIN_PASSWORD)

## API使用示例

```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "minimax2.7",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 技术栈

- 后端: Go + Gin
- 前端: Vue3 + Element Plus
- 数据库: SQLite
- 部署: Docker

## 项目结构

```
ai-gateway/
├── cmd/server/       # 入口文件
├── internal/         # 内部包
│   ├── config/       # 配置
│   ├── model/        # 数据模型
│   ├── handler/      # 控制器
│   ├── service/      # 业务逻辑
│   ├── middleware/   # 中间件
│   ├── repository/   # 数据访问
│   └── router/       # 路由
├── web/              # 前端代码
├── docs/             # 文档
└── docker-compose.yml
```
