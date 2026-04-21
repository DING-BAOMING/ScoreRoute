# ScoreRoute 开发工作流程

## 项目信息
- **项目路径**: /home/ubuntu/OpenCode/ai-gateway
- **分支**: main
- **域名**: https://api.029101.xyz
- **环境**: Docker (生产级)
- **登录凭证**: admin / dbm52100

## 快速开始

### 常用命令
```bash
# 构建并启动
cd /home/ubuntu/OpenCode/ai-gateway
sudo docker-compose down && sudo docker build -t ai-gateway-app:latest . && sudo docker-compose up -d

# 查看日志
sudo docker logs -f ai-gateway-app-1

# 健康检查
curl -s https://api.029101.xyz/health

# 登录获取token
curl -s -X POST "https://api.029101.xyz/api/auth/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"dbm52100"}'
```

## 核心功能

### 1. 模型选择流程
1. 用户创建Token时选择base model name (如 "MiniMax-M2")
2. API请求时发送 model 参数
3. Dispatcher 逻辑:
   - 先尝试 exact match (精确匹配)
   - 失败则尝试 prefix match (前缀匹配)
   - Smart模式: 选择最高分模型
   - Polling模式: 选择最低call_count模型

### 2. 调度模式
- **Smart (智能)**: 按评分选择最优模型
- **Polling (轮询)**: 按调用次数轮询

### 3. 特殊模型名称
| 名称 | 行为 |
|------|------|
| auto | 根据全局调度模式选择 (Smart/Polling) |
| AUTO/__AUTO__ | 使用GetRankedModelsSmart (总是智能调度) |
| __POLL_ALL__ | 轮询所有模型 |
| POLL_ALL | 同__POLL_ALL__ |
| 固定模型名 | 精确匹配或前缀匹配 |

## 最近修复 (2026-04-21)

### 1. 模型评分页筛选功能恢复
- **问题**: 添加Tab时筛选功能被移除
- **修复**: 恢复了以下筛选功能:
  - 按格式筛选 (openai, zhipu等)
  - 按类型筛选 (chat, embedding等)
  - 按格式+类型组合筛选
  - 按模型名称筛选
- **文件**: web/src/views/ModelRating.vue

### 2. Dispatcher自动模式修复
- **问题**: "auto"模式忽略全局调度模式设置
- **修复**: 
  - Polling + "auto" → GetNextModelGlobal (call_count轮询)
  - Smart + "auto" → GetRankedModelsSmart (评分选择)
- **文件**: internal/service/dispatcher.go

### 3. 模型名称标准化
- **问题**: minimaxai/minimax-m2.7 和 MiniMax-M2.7 被当作不同模型
- **修复**: 改进了normalizeModelName函数:
  - 去除常见提供商前缀 (minimaxai/, z-ai/, qwen/等)
  - 确保不同格式的同一模型被正确归一化
- **文件**: web/src/views/Tokens.vue

## 测试验证 (2026-04-21)

### 10次请求测试结果
| 组合 | 结果 | 说明 |
|------|------|------|
| Smart + auto | ✅ 10/10成功 | 选择最高分模型 |
| Smart + 固定模型 | ⚠️ 部分失败 | 上游API问题(529)，但前缀匹配正常 |
| Polling + auto | ✅ 10/10成功 | 轮询选择低call_count模型 |
| Polling + 固定模型 | ✅ 8/10成功 | 轮询正常，上游偶发故障 |

## 模型评分页面使用说明

### 筛选功能
1. 切换到"按格式类型"Tab可使用:
   - 筛选格式: 下拉选择如 openai
   - 筛选类型: 下拉选择如 chat
   - 格式+类型: 组合筛选如 openai_chat
   - 清除筛选: 重置所有筛选

2. 切换到"按模型"Tab可使用:
   - 筛选模型: 输入或选择模型名称
   - 清除筛选: 重置筛选

### 模型名称归一化
- 前缀如 "minimaxai/", "z-ai/" 等会被自动去除
- minimaxai/minimax-m2.7 和 MiniMax-M2.7 会被归一化为同一模型

## 文件修改记录

### 2026-04-21 修改

| 文件 | 修改内容 |
|------|----------|
| web/src/views/ModelRating.vue | 恢复筛选功能 |
| web/src/views/Tokens.vue | 改进模型名称标准化 |
| internal/service/dispatcher.go | 修复auto模式dispatch逻辑 |
| .session_state.md | 更新状态 |

## Git 提交记录
- 18 commits ahead of origin/main
- 最新commit: 9c82159 - fix: improve model name normalization in Tokens.vue

### GitHub PR
- fix/filter-and-dispatch: 筛选功能 + auto dispatch修复
- fix/model-normalization: 模型名称标准化

## 开发指南

### 添加新功能流程
1. 先读取 session_state.md 了解当前状态
2. 修改代码，一次只改一个文件
3. 本地构建测试: `go build ./cmd/server`
4. Docker构建部署: `sudo docker build -t ai-gateway-app:latest . && sudo docker-compose up -d`
5. 测试验证
6. 更新 session_state.md
7. Git提交

### 注意事项
- 每次最多修改一个文件
- 修改后必须测试
- 更新文档和session_state
- 提交前检查 git status
- 模型评分是动态变化的，根据实际性能计算
