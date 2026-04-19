# 开发计划

## 项目概述

ScoreRoute 是一个智能API网关，支持多渠道轮询、模型评分、自动启禁用等功能。

## 技术栈

- 后端: Go + Gin + SQLite
- 前端: Vue3 + Element Plus
- 部署: Docker

## 已完成功能

### 后端
- [x] 登录认证
- [x] 渠道管理 CRUD
- [x] 模型管理 CRUD
- [x] Token管理 CRUD
- [x] 调用日志
- [x] 轮询负载均衡
- [x] 流式响应 (SSE)
- [x] 健康检查
- [x] 定时日志清理
- [x] 模型评分统计
- [x] 用户评分系统
- [x] 自动启用/禁用

### 前端
- [x] 登录页面
- [x] 仪表盘
- [x] 接入管理
- [x] 模型管理
- [x] 接出管理
- [x] 日志查看
- [x] 模型评分页面
- [x] 用户评分页面
- [x] 开发文档页面

## 已接入渠道

| 渠道 | Provider | 状态 |
|------|----------|------|
| MiniMax-Production | MiniMax | ✅ |
| baoming | NVIDIA | ✅ |
| JDBook | NVIDIA | ✅ |
| baoming_ai | NVIDIA | ✅ |

## 架构说明

### 评分算法 (7因子)

| 指标 | 权重 | 说明 |
|------|------|------|
| 成功率 | 10% | 成功请求占总请求比例 |
| 延迟分数 | 10% | 延迟越低分数越高 |
| 稳定性 | 10% | 基于样本量 |
| 用户评分 | 20% | 用户1-100评分 |
| 样本评分 | 30% | 自动样本分析 |
| 成本评分 | 10% | 基于API成本 |
| 时间评分 | 10% | 基于过期时间 |

### 奖励/惩罚机制

- 新模型: +N分奖励(24小时衰减)
- 失败调用: -N分惩罚(每调用+1恢复)
- 惩罚低于0分自动删除

### 轮询策略

- 渠道优先 + 模型次之
- 按调用次数升序选择

## 详细开发日志

完整开发日志请参考Git提交历史:
```bash
git log --oneline
```

## 相关链接

- 官网: https://www.scoreroute.com
- 演示: https://demo.scoreroute.com  
- API: https://api.scoreroute.com
- 问题反馈: https://github.com/DING-BAOMING/ScoreRoute/issues
