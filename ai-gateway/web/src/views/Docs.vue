<template>
  <div class="docs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>开发文档</span>
        </div>
      </template>

      <div class="doc-list">
        <h2>用户文档</h2>
        <ul>
          <li><a href="/docs/getting-started.md" target="_blank">快速开始</a> - 5分钟快速上手</li>
          <li><a href="/docs/guide.md" target="_blank">用户指南</a> - 完整功能使用说明</li>
          <li><a href="/docs/api.md" target="_blank">API文档</a> - 接口说明和示例</li>
          <li><a href="/docs/faq.md" target="_blank">常见问题</a> - FAQ解答</li>
          <li><a href="/docs/changelog.md" target="_blank">更新日志</a> - 版本变更记录</li>
        </ul>

        <h2>开发者文档</h2>
        <ul>
          <li><a href="/docs/dev/setup.md" target="_blank">开发环境搭建</a> - 本地开发环境配置</li>
          <li><a href="/docs/dev/architecture.md" target="_blank">系统架构</a> - 技术架构说明</li>
          <li><a href="/docs/dev/plan.md" target="_blank">完整开发日志</a> - 开发进度和任务</li>
        </ul>

        <h2>快速链接</h2>
        <ul>
          <li>在线体验: <a href="https://api.029101.xyz" target="_blank">api.029101.xyz</a></li>
          <li>问题反馈: <a href="https://github.com/DING-BAOMING/ScoreRoute/issues" target="_blank">GitHub Issues</a></li>
        </ul>
      </div>
    </el-card>
  </div>
</template>

<script setup>
</script>

<style scoped>
.doc-list {
  line-height: 2;
}
.doc-list h2 {
  margin-top: 20px;
  margin-bottom: 10px;
  color: #409eff;
}
.doc-list ul {
  list-style: none;
  padding-left: 20px;
}
.doc-list a {
  color: #67c23a;
  text-decoration: none;
}
.doc-list a:hover {
  text-decoration: underline;
}
</style>
