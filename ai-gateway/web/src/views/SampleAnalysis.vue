<template>
  <div class="sample-analysis">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>样本分析</span>
          <el-button type="primary" @click="loadStats">
            <el-icon><Refresh /></el-icon>
            刷新统计
          </el-button>
        </div>
      </template>

      <el-row :gutter="20" style="margin-bottom: 20px">
        <el-col :span="6">
          <el-statistic title="总样本数" :value="stats.total" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="模型数" :value="stats.models" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="平均Token数" :value="stats.avgTokens" suffix="tokens" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="过期样本" :value="stats.expired" />
        </el-col>
      </el-row>

      <el-table :data="samples" v-loading="loading" stripe>
        <el-table-column prop="model_key" label="模型名称" width="200" />
        <el-table-column label="请求内容" min-width="200">
          <template #default="{ row }">
            <el-tooltip :content="row.request_content" placement="top" :show-after="300">
              <div class="content-preview">{{ row.request_content }}</div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="响应内容" min-width="200">
          <template #default="{ row }">
            <el-tooltip :content="row.response_content" placement="top" :show-after="300">
              <div class="content-preview">{{ row.response_content }}</div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="token_count" label="Token数" width="100" />
        <el-table-column label="剩余有效期" width="140">
          <template #default="{ row }">
            <el-tag :type="row.remaining_minutes <= 60 ? 'danger' : (row.remaining_minutes <= 1440 ? 'warning' : 'success')">
              {{ formatRemaining(row.remaining_minutes) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="viewDetail(row)">查看</el-button>
            <el-button size="small" type="warning" @click="analyze(row)">分析</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        style="margin-top: 20px"
        background
        layout="total, prev, pager, next"
        :total="total"
        :current-page="page"
        :page-size="pageSize"
        @current-change="loadSamples"
      />
    </el-card>

    <el-dialog v-model="detailVisible" title="样本详情" width="900px">
      <el-descriptions :column="2" border v-if="currentSample">
        <el-descriptions-item label="模型名称">{{ currentSample.model_key }}</el-descriptions-item>
        <el-descriptions-item label="Token数">{{ currentSample.token_count }}</el-descriptions-item>
        <el-descriptions-item label="剩余分钟数">{{ currentSample.remaining_minutes }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatTime(currentSample.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="过期时间">{{ formatTime(currentSample.expires_at) }}</el-descriptions-item>
      </el-descriptions>
      
      <el-tabs style="margin-top: 20px">
        <el-tab-pane label="请求内容">
          <pre class="content-pre">{{ currentSample?.request_content }}</pre>
        </el-tab-pane>
        <el-tab-pane label="响应内容">
          <pre class="content-pre">{{ currentSample?.response_content }}</pre>
        </el-tab-pane>
      </el-tabs>
      
      <template #footer>
        <el-button type="warning" @click="analyze(currentSample)">分析样本</el-button>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { sampleAPI } from '../api'

const loading = ref(false)
const samples = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const detailVisible = ref(false)
const currentSample = ref(null)

const stats = reactive({
  total: 0,
  models: 0,
  avgTokens: 0,
  expired: 0
})

onMounted(() => {
  loadSamples()
  loadStats()
})

async function loadSamples() {
  loading.value = true
  try {
    const res = await sampleAPI.list({ page: page.value, page_size: pageSize.value })
    if (res.code === 0) {
      samples.value = res.data || []
      total.value = samples.value.length
    }
  } catch (e) {
    ElMessage.error('加载样本失败')
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const res = await sampleAPI.stats()
    if (res.code === 0) {
      stats.total = res.data?.total_samples || 0
      stats.models = res.data?.models || 0
      stats.avgTokens = res.data?.avg_tokens || 0
      stats.expired = res.data?.expired || 0
    }
  } catch (e) {
    ElMessage.error('加载统计失败')
  }
}

function viewDetail(row) {
  currentSample.value = row
  detailVisible.value = true
}

function analyze(row) {
  ElMessage.info('分析功能开发中...')
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm('确定要删除该样本吗？此操作不可恢复', '警告', { type: 'warning' })
    await sampleAPI.delete(row.id)
    ElMessage.success('删除成功')
    loadSamples()
    loadStats()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function formatTime(time) {
  if (!time) return '-'
  const date = new Date(time)
  return date.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
}

function formatRemaining(minutes) {
  if (!minutes || minutes <= 0) return '已过期'
  if (minutes < 60) return `${minutes}分钟`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}小时${minutes % 60}分钟`
  const days = Math.floor(minutes / 1440)
  const hours = Math.floor((minutes % 1440) / 60)
  return `${days}天${hours}小时`
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.content-preview {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #666;
}

.content-pre {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 4px;
  max-height: 400px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
}
</style>
