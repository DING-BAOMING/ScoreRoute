<template>
  <div class="logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>调用日志</span>
          <div>
            <el-button type="info" @click="loadData" :loading="loading">
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
            <el-button type="danger" @click="handleCleanup">
              <el-icon><Delete /></el-icon>
              清理日志
            </el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" class="filter-form">
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            @change="loadData"
          />
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
      
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="token_name" label="接出API" width="150" />
        <el-table-column prop="channel_name" label="接入渠道" width="150" />
        <el-table-column prop="model_name" label="模型" />
        <el-table-column prop="latency_ms" label="延迟(ms)" width="100">
          <template #default="{ row }">
            <span :class="{ 'high-latency': row.latency_ms > 5000 }">
              {{ row.latency_ms }}ms
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="token_used" label="Token" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status < 400 ? 'success' : 'danger'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="error" label="错误信息" show-overflow-tooltip />
      </el-table>

      <el-pagination
        style="margin-top: 20px"
        background
        layout="total, prev, pager, next"
        :total="total"
        :current-page="page"
        :page-size="pageSize"
        @current-change="(p) => { page.value = p; loadData() }"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { logAPI } from '../api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const dateRange = ref([])
const refreshInterval = ref(null)

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadData()
  refreshInterval.value = setInterval(() => {
    if (page.value === 1 && (!dateRange.value || dateRange.value.length === 0)) {
      loadData()
    }
  }, 5000)
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})

async function loadData() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value
    }
    
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_time = dateRange.value[0]
      params.end_time = dateRange.value[1]
    }

    const res = await logAPI.list(params)
    if (res.code === 0) {
      list.value = res.data?.items || []
      total.value = res.data?.total || 0
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function resetFilter() {
  dateRange.value = []
  page.value = 1
  loadData()
}

async function handleCleanup() {
  await ElMessageBox.prompt('请输入保留天数', '清理日志', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    inputPlaceholder: '保留最近多少天的日志'
  }).then(async ({ value }) => {
    const days = parseInt(value)
    if (isNaN(days) || days <= 0) {
      ElMessage.warning('请输入有效的天数')
      return
    }
    
    try {
      await logAPI.cleanup(days)
      ElMessage.success('清理成功')
      loadData()
    } catch (e) {
      ElMessage.error('清理失败')
    }
  }).catch(() => {})
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.high-latency {
  color: #f56c6c;
  font-weight: bold;
}
</style>
