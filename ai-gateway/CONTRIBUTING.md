# Contributing to ScoreRoute

感谢您对 ScoreRoute 的关注！欢迎各种形式的贡献。

## 如何贡献

### 报告 Bug

1. 在 [Issues](https://github.com/DING-BAOMING/ScoreRoute/issues) 中创建新 issue
2. 选择 Bug 模板
3. 提供详细的重现步骤和预期/实际结果

### 提出功能建议

1. 在 [Issues](https://github.com/DING-BAOMING/ScoreRoute/issues) 中创建新 issue
2. 选择 Feature Request 模板
3. 详细描述功能需求和使用场景

### 提交代码

1. Fork 仓库
2. 创建特性分支: `git checkout -b feature/your-feature-name`
3. 提交更改: `git commit -m 'Add some feature'`
4. 推送分支: `git push origin feature/your-feature-name`
5. 创建 Pull Request

## 代码规范

### Go 代码

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 公共函数必须添加注释

### Vue 前端代码

- 遵循 Vue 3 最佳实践
- 使用 Composition API
- 组件文件使用 PascalCase 命名

### 提交信息格式

```
<type>: <subject>

<body>
```

**Type 类型:**
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例:**
```
feat: add model rating statistics page

Add a new page to display model rating details with:
- Success rate chart
- Latency distribution
- User ratings overview
```

## 开发环境

```bash
# 克隆仓库
git clone https://github.com/DING-BAOMING/ScoreRoute.git
cd ScoreRoute

# 复制环境配置
cp .env.example .env
# 编辑 .env 填入必要的环境变量

# 启动开发服务
docker-compose up -d
```

## Pull Request 流程

1. 确保所有测试通过
2. 更新相关文档
3. PR 描述清楚修改内容和原因
4. 等待维护者审核

## 许可证

贡献代码即表示您同意您的代码遵循项目的 MIT 许可证。
