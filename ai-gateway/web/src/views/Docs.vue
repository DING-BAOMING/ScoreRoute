<template>
  <div class="docs-page">
    <div class="docs-sidebar">
      <div class="sidebar-header">
        <span>📚 文档中心</span>
      </div>
      
      <div class="search-box">
        <el-input
          v-model="searchQuery"
          placeholder="搜索文档..."
          clearable
          size="small"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>

      <div class="doc-section">
        <div class="section-title">用户文档</div>
        <div
          v-for="doc in filteredUserDocs"
          :key="doc.path"
          class="doc-item"
          :class="{ active: currentDoc === doc.path }"
          @click="loadDoc(doc.path)"
        >
          {{ doc.name }}
        </div>
      </div>

      <div class="doc-section">
        <div class="section-title">开发者文档</div>
        <div
          v-for="doc in filteredDevDocs"
          :key="doc.path"
          class="doc-item"
          :class="{ active: currentDoc === doc.path }"
          @click="loadDoc(doc.path)"
        >
          {{ doc.name }}
        </div>
      </div>
    </div>

    <div class="docs-main" :class="{ 'has-content': docContent }">
      <div v-if="loading" class="loading">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>加载中...</span>
      </div>
      
      <div v-else-if="error" class="error">
        <p>加载文档失败: {{ error }}</p>
        <el-button @click="loadDoc('/docs/README.md')">返回首页</el-button>
      </div>
      
      <div v-else-if="docContent" class="doc-content">
        <pre class="markdown-content">{{ docContent }}</pre>
      </div>
      
      <div v-else class="welcome">
        <h1>ScoreRoute 文档中心</h1>
        <p>选择一个文档开始阅读</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Search, Loading } from '@element-plus/icons-vue'

const userDocs = [
  { name: '快速开始', path: '/docs/getting-started.md' },
  { name: '用户指南', path: '/docs/guide.md' },
  { name: 'API文档', path: '/docs/api.md' },
  { name: '常见问题', path: '/docs/faq.md' },
  { name: '更新日志', path: '/docs/changelog.md' }
]

const devDocs = [
  { name: '工作流程', path: '/docs/dev/workflow.md' },
  { name: '开发环境搭建', path: '/docs/dev/setup.md' },
  { name: '系统架构', path: '/docs/dev/architecture.md' },
  { name: '完整开发日志', path: '/docs/dev/plan_full.md' }
]

const currentDoc = ref('')
const docContent = ref('')
const loading = ref(false)
const error = ref('')
const searchQuery = ref('')

const filteredUserDocs = computed(() => {
  if (!searchQuery.value) return userDocs
  return userDocs.filter(d => d.name.includes(searchQuery.value))
})

const filteredDevDocs = computed(() => {
  if (!searchQuery.value) return devDocs
  return devDocs.filter(d => d.name.includes(searchQuery.value))
})

const loadDoc = async (path) => {
  loading.value = true
  error.value = ''
  currentDoc.value = path
  docContent.value = ''
  
  try {
    const response = await fetch(path)
    if (!response.ok) {
      throw new Error('HTTP ' + response.status)
    }
    const text = await response.text()
    docContent.value = text
  } catch (e) {
    error.value = e.message
    docContent.value = ''
  }
  
  loading.value = false
}

onMounted(() => {
  loadDoc('/docs/README.md')
})
</script>

<style scoped>
.docs-page {
  display: flex;
  min-height: calc(100vh - 60px);
  background: #ffffff;
}

.docs-sidebar {
  width: 260px;
  min-width: 260px;
  background: #ffffff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  font-weight: bold;
  font-size: 16px;
  color: #303133;
  background: #ffffff;
}

.search-box {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #ffffff;
}

.doc-section {
  padding: 12px 0;
  background: #ffffff;
}

.section-title {
  padding: 8px 16px;
  font-size: 12px;
  color: #909399;
  text-transform: uppercase;
}

.doc-item {
  padding: 10px 16px 10px 24px;
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  transition: all 0.2s;
  background: #ffffff;
}

.doc-item:hover {
  background: #ecf5ff;
  color: #409eff;
}

.doc-item.active {
  background: #409eff;
  color: #ffffff;
}

.docs-main {
  flex: 1;
  overflow-y: auto;
  background: #ffffff;
  padding: 30px 50px;
}

.docs-main.has-content {
  background: #ffffff;
}

.loading, .error, .welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #606266;
  background: #ffffff;
}

.loading .el-icon {
  font-size: 32px;
  margin-bottom: 10px;
}

.error {
  color: #f56c6c;
  background: #ffffff;
}

.doc-content {
  max-width: 900px;
  margin: 0 auto;
  background: #ffffff;
}

.markdown-content {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  font-size: 15px;
  line-height: 1.8;
  color: #24292e;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: #ffffff;
}

.welcome h1 {
  font-size: 28px;
  color: #303133;
}

.welcome p {
  color: #909399;
  margin-top: 10px;
}
</style>
