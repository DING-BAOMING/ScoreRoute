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
  - 处理 mini-max -> minimax 等变体
  - 确保不同格式的同一模型被正确归一化
- **文件**: 
  - internal/service/dispatcher_utils.go (normalizeModelNameForPrefix函数)
  - internal/service/dispatcher.go (使用标准化)

## 测试验证 (2026-04-21) - 10次请求测试结果

| 组合 | 结果 | 说明 |
|------|------|------|
| Smart + __AUTO__ | ✅ 10/10成功 | 正确选择最高分模型 |
| Smart + minimax-m2.5 | ✅ 10/10成功 | 正确匹配并调度 |
| Polling + __AUTO__ | ✅ 10/10成功 | 正确轮询低call_count模型 |
| Polling + minimax-m2.5 | ✅ 8/10成功 | 上游504/529错误,调度器fallback正常 |

### Demo-1 API测试
- Token: sk-c0c45ba8-2657-4d12-be5c-912c88857739
- 测试: 5/5 成功请求
- 状态: ✅ 正常工作

## Git 状态
- **main分支**: 23 commits ahead of origin/main
- **已推送分支**:
  - fix/filter-and-dispatch
  - fix/model-normalization
  - sync-docs-0421
  - fix/model-name-normalization

### GitHub PR 链接
由于main分支有保护规则,需要通过PR合并:
- https://github.com/DING-BAOMING/ScoreRoute/pull/new/fix/filter-and-dispatch
- https://github.com/DING-BAOMING/ScoreRoute/pull/new/fix/model-normalization
- https://github.com/DING-BAOMING/ScoreRoute/pull/new/sync-docs-0421
- https://github.com/DING-BAOMING/ScoreRoute/pull/new/fix/model-name-normalization

## 服务状态
- 健康检查: ✅ 正常
- Docker镜像: 77b9f983d9bb
- 当前调度模式: polling

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
- mini-max-m2.5 会被归一化为 minimax-m2.5

## 评分系统详解 (2026-04-21)

### 评分规则

| 评分项 | 权重 | 计算公式 |
|--------|------|----------|
| 成功率 | 15% | successCalls/totalCalls*100 |
| 延迟分数 | 10% | max(0, 1-avgLatency/30000)*100 |
| 稳定性 | 10% | 基于样本量: >=30次=100分 |
| 用户评分 | 15% | 默认85，来自user_ratings表 |
| 样本评分 | 25% | 默认85，来自sample_ratings表 |
| 成本评分 | 15% | 免费=90，定期=100 |
| 时效评分 | 10% | 接近过期=100 |
| 惩罚/奖励 | 加分项 | 来自extra_ratings表 |

### 实时更新机制
每次API调用后:
1. 日志保存到call_logs表
2. 调度器调用CalculateAllScores()
3. GetModelStatsMap()聚合调用统计
4. 分数实时重新计算

### 测试验证 (2026-04-21)
- Smart + __AUTO__: ✅ 10/10
- Smart + minimax-m2.5: ✅ 10/10
- Polling + __AUTO__: ✅ 10/10
- Polling + minimax-m2.5: ✅ 10/10

## 问题排查记录 (2026-04-21)

### 1. baoming API Key 问题
- **现象**: NVIDIA API key 可以列出模型但无法进行chat completions
- **错误**: `{"error":{"message":"404 Function not found for account..."}}`
- **原因**: API key 权限限制，仅允许模型列表操作
- **状态**: 上游API key限制，非代码问题

### 2. baoming 速率限制更新
- **操作**: 直接更新数据库
- **旧限制**: `{"type":"calls","max_count":40,"window":"minute"}`
- **新限制**: `{"type":"tokens","max_count":1000000000,"window":"hour"}`
- **SQL**: `UPDATE channels SET rate_limits = '[{"type":"tokens","max_count":1000000000,"window":"hour"}]' WHERE id = 10;`

### 3. API 响应时间
- **现象**: NVIDIA API (baoming) 响应时间3分钟以上
- **影响**: 调度测试受限
- **状态**: 系统本身工作正常，上游API慢

### 4. Docs.vue 空白页面
- **现象**: 内嵌文档页面空白
- **排查**: 
  - 后端 `/docs/*` 正确返回markdown内容 ✅
  - JS包包含 marked 和 DOMPurify ✅
  - 需浏览器环境进一步调试
