<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.today_tokens || 0) }}</div>
              <div class="stat-label">今日Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a">
              <el-icon><Calendar /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.week_tokens || 0) }}</div>
              <div class="stat-label">本周Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #e6a23c">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.month_tokens || 0) }}</div>
              <div class="stat-label">本月Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #f56c6c">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ Math.round(stats.avg_latency || 0) }}ms</div>
              <div class="stat-label">平均延迟</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #909399">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ channelCount }}</div>
              <div class="stat-label">渠道数量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ tokenCount }}</div>
              <div class="stat-label">API数量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.today_calls || 0 }}</div>
              <div class="stat-label">今日调用</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #e6a23c">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.total_calls || 0 }}</div>
              <div class="stat-label">总调用次数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>渠道调用排行</span>
          </template>
          <el-table :data="topChannels" stripe>
            <el-table-column prop="channel_name" label="渠道名称" />
            <el-table-column prop="call_count" label="调用次数" width="100" />
            <el-table-column prop="avg_latency" label="平均延迟" width="100">
              <template #default="{ row }">
                {{ row.avg_latency ? Math.round(row.avg_latency) + 'ms' : '-' }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>API调用统计</span>
          </template>
          <el-table :data="tokenStats" stripe>
            <el-table-column prop="token_name" label="API名称" />
            <el-table-column prop="total_calls" label="总调用" width="80" />
            <el-table-column prop="today_calls" label="今日" width="70" />
            <el-table-column prop="week_calls" label="本周" width="70" />
            <el-table-column prop="month_calls" label="本月" width="70" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { logAPI, channelAPI, tokenAPI } from '../api'

const stats = ref({})
const topChannels = ref([])
const tokenStats = ref([])
const channelCount = ref(0)
const tokenCount = ref(0)

function formatNumber(num) {
  if (!num) return '0'
  if (num >= 100000000) {
    return (num / 100000000).toFixed(1).replace(/\.0$/, '') + '亿'
  }
  if (num >= 10000) {
    return (num / 10000).toFixed(1).replace(/\.0$/, '') + '万'
  }
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

onMounted(async () => {
  await loadDashboard()
})

async function loadDashboard() {
  try {
    const [dashboardRes, channelsRes, tokensRes] = await Promise.all([
      logAPI.dashboard(),
      channelAPI.list({ page: 1, page_size: 1 }),
      tokenAPI.list({ page: 1, page_size: 1 })
    ])
    
    if (dashboardRes.code === 0) {
      stats.value = dashboardRes.data?.items?.stats || {}
      topChannels.value = dashboardRes.data?.items?.top_channels || []
      tokenStats.value = dashboardRes.data?.items?.token_stats || []
    }
    
    if (channelsRes.code === 0) {
      channelCount.value = channelsRes.data?.total || 0
    }

    if (tokensRes.code === 0) {
      tokenCount.value = tokensRes.data?.total || 0
    }
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 20px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 24px;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #333;
}

.stat-label {
  color: #999;
  font-size: 14px;
  margin-top: 5px;
}

</style>
