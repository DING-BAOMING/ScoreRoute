<template>
  <div class="docs-container">
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

    <div class="docs-main">
      <div v-if="loading" class="loading">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>加载中...</span>
      </div>
      
      <div v-else-if="error" class="error">
        <p>加载文档失败: {{ error }}</p>
        <el-button @click="loadDoc('/docs/README.md')">返回首页</el-button>
      </div>
      
      <div v-else-if="renderedContent" class="markdown-content" v-html="renderedContent"></div>
      
      <div v-else class="welcome">
        <h1>ScoreRoute 文档中心</h1>
        <p>选择一个文档开始阅读</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { marked } from 'marked'
import { Search, Loading } from '@element-plus/icons-vue'

const userDocs = [
  { name: '快速开始', path: '/docs/getting-started.md' },
  { name: '用户指南', path: '/docs/guide.md' },
  { name: 'API文档', path: '/docs/api.md' },
  { name: '常见问题', path: '/docs/faq.md' },
  { name: '更新日志', path: '/docs/changelog.md' }
]

const devDocs = [
  { name: '开发环境搭建', path: '/docs/dev/setup.md' },
  { name: '系统架构', path: '/docs/dev/architecture.md' },
  { name: '完整开发日志', path: '/docs/dev/plan_full.md' }
]

const allDocs = [...userDocs, ...devDocs]

const currentDoc = ref('')
const renderedContent = ref('')
const loading = ref(false)
const error = ref('')
const searchQuery = ref('')

const filteredUserDocs = computed(() => {
  if (!searchQuery.value) return userDocs
  const q = searchQuery.value.toLowerCase()
  return userDocs.filter(d => d.name.toLowerCase().includes(q))
})

const filteredDevDocs = computed(() => {
  if (!searchQuery.value) return devDocs
  const q = searchQuery.value.toLowerCase()
  return devDocs.filter(d => d.name.toLowerCase().includes(q))
})

const loadDoc = async (path) => {
  loading.value = true
  error.value = ''
  currentDoc.value = path
  
  try {
    const response = await fetch(path)
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    const text = await response.text()
    renderedContent.value = marked(text)
  } catch (e) {
    error.value = e.message
    renderedContent.value = ''
  }
  
  loading.value = false
}

onMounted(() => {
  loadDoc('/docs/README.md')
})
</script>

<style scoped>
.docs-container {
  display: flex;
  height: calc(100vh - 60px);
  background: #f5f7fa;
}

.docs-sidebar {
  width: 260px;
  min-width: 260px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  font-weight: bold;
  font-size: 16px;
  color: #303133;
}

.search-box {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
}

.doc-section {
  padding: 12px 0;
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
}

.doc-item:hover {
  background: #ecf5ff;
  color: #409eff;
}

.doc-item.active {
  background: #409eff;
  color: #fff;
}

.docs-main {
  flex: 1;
  overflow-y: auto;
  background: #fff;
}

.loading, .error, .welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #606266;
}

.loading .el-icon {
  font-size: 32px;
  margin-bottom: 10px;
}

.error {
  color: #f56c6c;
}

.markdown-content {
  padding: 30px 50px;
  max-width: 1000px;
  margin: 0 auto;
  line-height: 1.8;
}

.markdown-content :deep(h1) {
  font-size: 28px;
  color: #303133;
  border-bottom: 2px solid #409eff;
  padding-bottom: 12px;
  margin-bottom: 20px;
}

.markdown-content :deep(h2) {
  font-size: 22px;
  color: #303133;
  margin-top: 30px;
  margin-bottom: 15px;
  border-left: 4px solid #409eff;
  padding-left: 12px;
}

.markdown-content :deep(h3) {
  font-size: 18px;
  color: #606266;
  margin-top: 20px;
  margin-bottom: 10px;
}

.markdown-content :deep(p) {
  margin: 12px 0;
  color: #606266;
}

.markdown-content :deep(code) {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: Monaco, Consolas, monospace;
  font-size: 14px;
  color: #e6a23c;
}

.markdown-content :deep(pre) {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 15px 0;
}

.markdown-content :deep(pre code) {
  background: none;
  padding: 0;
  color: #303133;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  padding-left: 24px;
  margin: 10px 0;
}

.markdown-content :deep(li) {
  margin: 6px 0;
}

.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 15px 0;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid #e4e7ed;
  padding: 10px 12px;
  text-align: left;
}

.markdown-content :deep(th) {
  background: #f5f7fa;
  font-weight: bold;
}

.markdown-content :deep(a) {
  color: #409eff;
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(blockquote) {
  border-left: 4px solid #67c23a;
  padding: 10px 15px;
  margin: 15px 0;
  background: #f0f9eb;
  color: #606266;
}

.markdown-content :deep(strong) {
  color: #303133;
}

.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid #e4e7ed;
  margin: 20px 0;
}
</style>
