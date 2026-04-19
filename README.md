# ScoreRoute

[![CI](https://github.com/DING-BAOMING/ScoreRoute/actions/workflows/ci.yml/badge.svg)](https://github.com/DING-BAOMING/ScoreRoute/actions)
[![Go Version](https://img.shields.io/badge/Go-1.23-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

开源的智能AI路由网关，支持GPT、Claude、Gemini等主流模型的多模型统一接入。通过智能评分、负载均衡、故障转移，让AI应用更稳定、更高效。

## 核心特性

### 🚀 多模型路由
- 支持GPT、Claude、Gemini、MiniMax等多种AI模型统一接入
- 自定义路由策略：随机、轮询、加权、最小延迟等

### ⚖️ 负载均衡
- 自动分发请求到多个模型实例
- 连接池管理、请求队列、限流熔断
- 有效避免单点故障和过载

### 🛡️ 故障转移
- 模型失败时自动切换，保障服务连续性
- 重试机制、自动跳过不可用模型
- SLA高达99.9%

### 🧠 智能调度
根据综合评分自动选择最优模型：
| 维度 | 权重 | 说明 |
|------|------|------|
| 样本评分 | 25% | 样本分析得出的评分 |
| 成功率 | 15% | 成功请求占总请求的比例 |
| 成本评分 | 15% | 免费模型90分，付费模型70分 |
| 用户评分 | 15% | 用户对模型的1-100评分 |
| 延迟分数 | 10% | 延迟越低分数越高 |
| 稳定性 | 10% | 样本量越多评分越可靠 |
| 时间评分 | 10% | 基于模型过期时间 |

### 📊 模型评分
多维度量化模型表现：
| 维度 | 权重 |
|------|------|
| 成功率 | 28% |
| 延迟分数 | 21% |
| 稳定性 | 21% |
| 用户评分 | 15% |
| 样本分析 | 15% |

### 🔬 样本分析
- 保存响应Token≥500的请求样本7天
- 每2小时自动分析最多20个样本
- 评估Agent场景下的工具调用、完整性、上下文理解等能力

### ⭐ 附加评分机制
- **惩罚机制**：连续调用某模型时施加-5分/次惩罚
- **奖励机制**：新模型加入时获得+5分奖励，持续24小时

### 🔐 API Key管理
- 安全的Key存储与轮换
- 多个API密钥集中管理
- 加密存储、最小权限

## 快速开始

### 一键安装

```bash
curl -O https://raw.githubusercontent.com/DING-BAOMING/ScoreRoute/main/install.sh && chmod +x install.sh && ./install.sh
```

安装脚本会自动：
- 检测端口占用情况（默认8080）
- 配置系统服务
- 启动并验证服务

### Docker 部署

```bash
# 构建镜像
docker build -t scoreroute .

# 运行容器
docker run -d -p 8080:8080 \
  -e ADMIN_PASSWORD=your_password \
  -e JWT_SECRET=your_jwt_secret \
  -v ./data:/app/data \
  scoreroute
```

### 访问管理界面

安装完成后，访问 `http://your-server:8080`

默认端口：8080（可在安装时修改）

## API 使用

### Chat Completions API

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_KEY" \
  -d '{
    "model": "minimaxai/minimax-m2.5",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### 轮询所有模型

使用 `AUTO` 或 `POLL_ALL` 作为模型名，系统会自动轮询所有可用模型：

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_KEY" \
  -d '{
    "model": "AUTO",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### List Models API

```bash
curl -X GET http://localhost:8080/v1/models \
  -H "Authorization: Bearer YOUR_TOKEN_KEY"
```

## 技术栈

- **后端**: Go 1.23 + Gin
- **前端**: Vue 3 + Element Plus
- **数据库**: SQLite
- **部署**: Docker

## 项目结构

```
ai-gateway/
├── cmd/                 # 主程序入口
├── internal/            # 内部包
│   ├── config/         # 配置管理
│   ├── handler/        # HTTP处理器
│   ├── middleware/    # 中间件
│   ├── model/          # 数据模型
│   ├── repository/    # 数据访问层
│   └── service/        # 业务逻辑
├── web/                # Vue前端
│   └── src/
│       ├── views/      # 页面视图
│       ├── components/ # 组件
│       └── stores/     # 状态管理
├── install.sh          # 一键安装脚本
├── Dockerfile
└── docker-compose.yml
```

## 管理功能

### 渠道管理
- 支持多种AI服务商：OpenAI、Anthropic、Azure、Google、智谱Zhipu等
- 设置总Token限制、API有效期、调用频率限制

### 模型管理
- 批量添加模型
- 设置模型Token限制、过期时间、调用频率
- 配置每Token费用和货币单位

### Token管理
- 生成API访问凭证
- 绑定特定格式和模型
- 支持重新生成Key

### 样本分析
- 按剩余时间升序自动分析
- 分析完成后自动删除样本
- 评分保存7天，过期自动清理

## 常见问题

**Q: 调用返回401错误?**
A: 请检查API Key是否正确，以及Token是否被启用。

**Q: 调用返回429错误?**
A: 请求过于频繁，可能原因：
- 渠道设置了调用频率限制
- 渠道设置了Token总量限制
- 渠道API已过期
- 模型设置了调用频率限制

**Q: 如何添加新的AI渠道?**
A: 1. 在渠道管理中添加新的渠道配置
   2. 在模型管理中添加该渠道下的模型
   3. 在Token管理中创建访问凭证

## License

MIT License - 由 [BOMING](https://github.com/DING-BAOMING)开发和维护

## 联系方式

- 邮箱: [Admin@ScoreRoute.com](mailto:Admin@ScoreRoute.com)
- 官网: [https://www.scoreroute.com](https://www.scoreroute.com)
- 飞书群: [https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=82bq51e8-fbd2-4c36-97d9-4e9f575c3d1b](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=82bq51e8-fbd2-4c36-97d9-4e9f575c3d1b)
