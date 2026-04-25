# API文档

## 基础信息

- **API地址**: https://api.scoreroute.com
- **认证方式**: Bearer Token

## 认证

所有API需要通过以下方式认证：

```bash
-H "Authorization: Bearer YOUR_TOKEN_HERE"
```

登录获取Token:
```bash
curl -X POST "https://api.scoreroute.com/api/auth/login" \
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
curl -X POST "https://api.scoreroute.com/v1/chat/completions" \
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
curl -X POST "https://api.scoreroute.com/v1/chat/completions" \
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
| PUT | /api/channels/:id/enabled | 启用/禁用渠道 |

### 模型管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/models | 获取模型列表 |
| POST | /api/models | 创建模型 |
| PUT | /api/models/:id | 更新模型 |
| DELETE | /api/models/:id | 删除模型 |
| POST | /api/models/test/:id | 测试模型 |
| GET | /api/channels/:id/models | 获取渠道可用模型 |

### Token管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/tokens | 获取Token列表 |
| POST | /api/tokens | 创建Token |
| PUT | /api/tokens/:id | 更新Token |
| DELETE | /api/tokens/:id | 删除Token |
| PUT | /api/tokens/:id/enabled | 启用/禁用Token |

### 系统配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/system-config | 获取系统配置 |
| PUT | /api/system-config/dispatch-mode | 更新调度模式 (smart/polling) |
| PUT | /api/system-config/password-less | 开启/关闭免密模式 |


### 评分相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/model-rating/scores | 获取模型评分 |
| GET | /api/user-ratings | 获取用户评分 |
| POST | /api/user-ratings | 创建/更新用户评分 (参数: model_name, user_rating) |
| DELETE | /api/user-ratings/:id | 删除用户评分 |

### 日志

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/logs | 获取日志列表 |
| GET | /api/logs/stats | 获取统计信息 |
| GET | /api/model-rating/cost-time | 获取成本延迟评分 |

## 轮询机制

### 轮询策略
- **轮询方式**: 渠道优先 + 模型次之
- **排序规则**: 按调用次数升序选择

### 请求模型名称
| 模型名称 | 行为 |
|---------|------|
| `auto` | 按Token的format+type轮询 |
| `AUTO` | 轮询所有模型 |
| `__POLL_ALL__` | 轮询所有模型 |
| `__AUTO__` | 按format+type轮询 |
