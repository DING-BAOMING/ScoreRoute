<template>
  <div class="docs-page">
    <div class="docs-sidebar">
      <div class="sidebar-header">
        <span>文档中心</span>
        <el-button size="small" @click="showSearch = !showSearch">
          <el-icon><Search /></el-icon>
        </el-button>
      </div>
      
      <el-input
        v-if="showSearch"
        v-model="searchQuery"
        placeholder="搜索文档..."
        clearable
        @input="filterDocs"
      />

      <el-collapse v-model="activeSection" @change="handleSectionChange">
        <el-collapse-item title="用户文档" name="user">
          <div 
            v-for="doc in filteredUserDocs" 
            :key="doc.path"
            class="doc-item"
            :class="{ active: currentDoc === doc.path }"
            @click="loadDoc(doc.path)"
          >
            {{ doc.name }}
          </div>
        </el-collapse-item>
        
        <el-collapse-item title="开发者文档" name="dev">
          <div 
            v-for="doc in filteredDevDocs" 
            :key="doc.path"
            class="doc-item"
            :class="{ active: currentDoc === doc.path }"
            @click="loadDoc(doc.path)"
          >
            {{ doc.name }}
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>

    <div class="docs-content" v-loading="loading">
      <div v-if="!currentDoc" class="welcome">
        <h1>ScoreRoute 文档中心</h1>
        <p>选择一个文档开始阅读</p>
      </div>
      <div v-else class="markdown-body" v-html="renderedContent"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { marked } from 'marked'
import { Search } from '@element-plus/icons-vue'

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

const activeSection = ref(['user', 'dev'])
const currentDoc = ref('')
const renderedContent = ref('')
const loading = ref(false)
const showSearch = ref(false)
const searchQuery = ref('')

const filteredUserDocs = computed(() => {
  if (!searchQuery.value) return userDocs
  return userDocs.filter(d => d.name.includes(searchQuery.value))
})

const filteredDevDocs = computed(() => {
  if (!searchQuery.value) return devDocs
  return devDocs.filter(d => d.name.includes(searchQuery.value))
})

const filterDocs = () => {}

const loadDoc = async (path) => {
  loading.value = true
  currentDoc.value = path
  try {
    const response = await fetch(path)
    const text = await response.text()
    renderedContent.value = marked(text)
  } catch (error) {
    renderedContent.value = '<p>加载文档失败</p>'
  }
  loading.value = false
}

const handleSectionChange = () => {}

onMounted(() => {
  loadDoc('/docs/README.md')
})
</script>

<style scoped>
.docs-page {
  display: flex;
  height: calc(100vh - 60px);
  background: #f5f7fa;
}

.docs-sidebar {
  width: 280px;
  background: white;
  border-right: 1px solid #e4e7ed;
  overflow-y: auto;
  padding-bottom: 20px;
}

.sidebar-header {
  padding: 15px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
}

.doc-item {
  padding: 10px 20px;
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  transition: all 0.3s;
}

.doc-item:hover {
  background: #f5f7fa;
  color: #409eff;
}

.doc-item.active {
  background: #ecf5ff;
  color: #409eff;
  border-right: 3px solid #409eff;
}

.docs-content {
  flex: 1;
  overflow-y: auto;
  padding: 30px 50px;
  background: white;
}

.welcome {
  text-align: center;
  margin-top: 100px;
  color: #909399;
}

.markdown-body {
  max-width: 900px;
  margin: 0 auto;
  line-height: 1.8;
}

.markdown-body :deep(h1) {
  border-bottom: 2px solid #409eff;
  padding-bottom: 10px;
  margin-bottom: 20px;
  color: #303133;
}

.markdown-body :deep(h2) {
  margin-top: 30px;
  margin-bottom: 15px;
  color: #303133;
  border-left: 4px solid #409eff;
  padding-left: 10px;
}

.markdown-body :deep(h3) {
  margin-top: 20px;
  margin-bottom: 10px;
  color: #606266;
}

.markdown-body :deep(code) {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: Monaco, Consolas, monospace;
}

.markdown-body :deep(pre) {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 8px;
  overflow-x: auto;
}

.markdown-body :deep(pre code) {
  background: none;
  padding: 0;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 15px 0;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid #e4e7ed;
  padding: 10px;
  text-align: left;
}

.markdown-body :deep(th) {
  background: #f5f7fa;
  font-weight: bold;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 25px;
}

.markdown-body :deep(li) {
  margin: 5px 0;
}

.markdown-body :deep(a) {
  color: #409eff;
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(blockquote) {
  border-left: 4px solid #67c23a;
  padding: 10px 15px;
  margin: 15px 0;
  background: #f0f9eb;
  color: #606266;
}

.markdown-body :deep(strong) {
  color: #303133;
}
</style>
