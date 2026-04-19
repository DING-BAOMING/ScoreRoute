# 快速开始

## 5分钟快速上手

### 1. 获取API Key

登录系统后，进入"接出管理"页面创建Token。

### 2. 调用API

```bash
curl -X POST "https://api.029101.xyz/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'
```

### 3. 使用特定模型

```bash
# 使用指定模型
"model": "MiniMax-M2.5"

# 自动选择最优模型（推荐）
"model": "auto"
```

## 下一步

- 查看[用户指南](guide.md)了解更多功能
- 查看[API文档](api.md)了解接口详情
