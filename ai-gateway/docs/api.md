# API文档

## 认证

所有API需要通过以下方式认证：

```bash
-H "Authorization: Bearer YOUR_TOKEN_HERE"
```

登录获取Token:
```bash
curl -X POST "https://api.029101.xyz/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"YOUR_PASSWORD"}'
```

## 对外API

### 聊天完成

```
POST /v1/chat/completions
```

**请求示例:**
```bash
curl -X POST "https://api.029101.xyz/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'
```

**model参数说明:**
| 值 | 说明 |
|----|------|
| `auto` | 自动选择最优模型 |
| `MiniMax-M2.5` | 使用指定模型 |
| `__POLL_ALL__` | 轮询所有模型 |

### 流式响应

添加 `"stream": true` 参数：

```bash
curl -X POST "https://api.029101.xyz/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100,
    "stream": true
  }'
```

## 管理API

### 渠道管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/channels | 获取渠道列表 |
| POST | /api/channels | 创建渠道 |
| PUT | /api/channels/:id | 更新渠道 |
| DELETE | /api/channels/:id | 删除渠道 |
| POST | /api/channels/test-credentials | 测试渠道连接 |

### 模型管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/models | 获取模型列表 |
| POST | /api/models | 创建模型 |
| PUT | /api/models/:id | 更新模型 |
| DELETE | /api/models/:id | 删除模型 |
| POST | /api/models/test/:id | 测试模型 |

### Token管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/tokens | 获取Token列表 |
| POST | /api/tokens | 创建Token |
| PUT | /api/tokens/:id | 更新Token |
| DELETE | /api/tokens/:id | 删除Token |

### 评分相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/model-rating/scores | 获取模型评分 |
| GET | /api/user-ratings | 获取用户评分 |
| POST | /api/user-ratings | 创建/更新用户评分 |

### 日志

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/logs | 获取日志列表 |
| GET | /api/logs/stats | 获取统计信息 |
