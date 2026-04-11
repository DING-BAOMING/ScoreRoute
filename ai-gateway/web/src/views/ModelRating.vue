<template>
  <div class="model-rating">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>模型评分</span>
          <el-button type="primary" @click="loadData">
            <el-icon><RefreshRight /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-alert type="info" :closable="false" style="margin-bottom: 20px">
        <template #title>
          <span>评分说明</span>
        </template>
        <template #default>
          <ul style="margin: 5px 0 0 0; padding-left: 20px">
            <li><b>成功率</b> (28%): 成功请求占总请求的比例</li>
            <li><b>延迟分数</b> (21%): 基于平均延迟计算，延迟越低分数越高</li>
            <li><b>稳定性</b> (21%): 基于样本量计算，样本越多评分越可靠</li>
            <li><b>用户评分</b> (15%): 用户对模型的评分（1-100标准化）</li>
            <li><b>样本分析</b> (15%): 基于样本分析评分，评估工具调用、完整性等</li>
          </ul>
        </template>
      </el-alert>
      
      <el-table :data="displayList" v-loading="loading" stripe>
        <el-table-column prop="rank" label="排名" width="80">
          <template #default="{ row }">
            <span v-if="row.rank <= 3" :class="'rank rank-' + row.rank">{{ row.rank }}</span>
            <span v-else>{{ row.rank }}</span>
          </template>
        </el-table-column>
        <el-table-column label="模型" min-width="250">
          <template #default="{ row }">
            <div class="model-name">{{ row.channel_name }}/{{ row.format }}/{{ row.type }}/{{ row.model_name }}</div>
            <div v-if="row.user_rating" class="user-rating-badge">用户评分: {{ row.user_rating }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="total_calls" label="总调用" width="100" sortable>
          <template #default="{ row }">
            {{ formatNumber(row.total_calls) }}
          </template>
        </el-table-column>
        <el-table-column label="成功率" width="120" sortable>
          <template #default="{ row }">
            <el-progress :percentage="Math.round(row.success_rate)" 
              :color="getSuccessRateColor(row.success_rate)"
              :status="row.success_rate >= 95 ? 'success' : undefined" />
          </template>
        </el-table-column>
        <el-table-column prop="avg_latency" label="平均延迟" width="120" sortable>
          <template #default="{ row }">
            {{ Math.round(row.avg_latency) }}ms
          </template>
        </el-table-column>
        <el-table-column label="评分" width="100" sortable prop="score">
          <template #default="{ row }">
            <el-tag :type="getScoreType(row.score)" size="large">
              {{ row.score.toFixed(1) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="详情" width="200">
          <template #default="{ row }">
            <div class="detail-info">
              <span>成功: {{ row.success_calls }}</span>
              <span>失败: {{ row.failed_calls }}</span>
            </div>
            <div class="detail-info">
              <span>Token: {{ formatNumber(row.total_tokens) }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { logAPI, userRatingAPI, sampleAnalysisAPI } from '../api'

const loading = ref(false)
const modelStats = ref([])
const userRatings = ref({})
const sampleRatings = ref({})

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const [statsRes, ratingsRes, sampleRatingsRes] = await Promise.all([
      logAPI.modelStats(),
      userRatingAPI.listDeduplicated(),
      sampleAnalysisAPI.getRatingsMap()
    ])
    
    if (statsRes.code === 0) {
      const ratingsMap = {}
      if (ratingsRes.code === 0) {
        ;(ratingsRes.data || []).forEach(r => {
          ratingsMap[r.model_name.toLowerCase()] = r.user_rating
        })
      }
      userRatings.value = ratingsMap
      
      const sampleRatingsMap = {}
      if (sampleRatingsRes.code === 0) {
        ;(Object.values(sampleRatingsRes.data || {}) || []).forEach(r => {
          sampleRatingsMap[r.model_key.toLowerCase()] = r.score
        })
      }
      sampleRatings.value = sampleRatingsMap
      
      const stats = statsRes.data || []
      const scored = stats.map(s => {
        const userRating = getUserRatingForModel(s.model_name)
        const sampleRating = getSampleRatingForModel(s.model_name)
        const score = calculateScore(s, userRating, sampleRating)
        return { ...s, score, user_rating: userRating, sample_rating: sampleRating }
      })
      scored.sort((a, b) => b.score - a.score)
      scored.forEach((s, i) => s.rank = i + 1)
      modelStats.value = scored
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function getUserRatingForModel(modelName) {
  if (!modelName) return 50
  const normalized = normalizeModelName(modelName)
  return userRatings.value[normalized.toLowerCase()] || 50
}

function getSampleRatingForModel(modelName) {
  if (!modelName) return 0
  const normalized = normalizeModelName(modelName)
  return sampleRatings.value[normalized.toLowerCase()] || 0
}

function normalizeModelName(name) {
  const lower = name.toLowerCase()
  
  if (lower.includes('minimaxai/') || lower.includes('minimax-') || lower === 'minimax') {
    return normalizeMinimaxModel(name)
  }
  
  const vendorPrefixes = ['google/', 'qwen/', 'z-ai/', 'anthropic/', 'openai/', 'meta/', 'mistral/', 'cohere/', 'azure/', 'aws/', 'alibaba/', 'baidu/', 'tencent/']
  let normalized = lower
  for (const prefix of vendorPrefixes) {
    if (normalized.includes(prefix)) {
      normalized = normalized.replace(prefix, '')
      break
    }
  }
  return normalized
}

function normalizeMinimaxModel(name) {
  const lower = name.toLowerCase()
  if (lower.includes('minimaxai/')) {
    let n = name.toLowerCase().replace('minimaxai/', '')
    n = n.replace('minimax-', '').replace('minimax', '')
    if (n.startsWith('m') && n.length > 1 && n[1] >= '0' && n[1] <= '9') {
      n = n.substring(1)
    }
    return 'minimax-' + n
  }
  if (lower.includes('minimax-') || lower.includes('minimax')) {
    let n = name.toLowerCase().replace('minimax-', '').replace('minimax', '')
    if (n.startsWith('m') && n.length > 1 && n[1] >= '0' && n[1] <= '9') {
      n = n.substring(1)
    }
    return 'minimax-' + n
  }
  return lower
}

function calculateScore(stat, userRating, sampleRating) {
  const successWeight = 0.28
  const latencyWeight = 0.21
  const reliabilityWeight = 0.21
  const userRatingWeight = 0.15
  const sampleRatingWeight = 0.15

  const successRate = stat.success_rate / 100

  const maxLatencyThreshold = 30000
  const latencyScore = Math.max(0, 1 - (stat.avg_latency / maxLatencyThreshold))

  let reliabilityScore = 1
  if (stat.total_calls < 5) {
    reliabilityScore = stat.total_calls / 5 * 0.5
  } else if (stat.total_calls < 10) {
    reliabilityScore = 0.5 + (stat.total_calls - 5) / 10 * 0.3
  } else if (stat.total_calls < 30) {
    reliabilityScore = 0.8 + (stat.total_calls - 10) / 20 * 0.2
  } else {
    reliabilityScore = 1
  }

  const normalizedUserRating = userRating / 100
  const normalizedSampleRating = sampleRating / 100

  return successRate * successWeight + latencyScore * latencyWeight + reliabilityScore * reliabilityWeight + normalizedUserRating * userRatingWeight + normalizedSampleRating * sampleRatingWeight
}

const displayList = computed(() => {
  return modelStats.value
})

function getSuccessRateColor(rate) {
  if (rate >= 95) return '#67c23a'
  if (rate >= 80) return '#e6a23c'
  return '#f56c6c'
}

function getScoreType(score) {
  if (score >= 80) return 'success'
  if (score >= 60) return 'warning'
  return 'danger'
}

function formatNumber(num) {
  if (!num) return '0'
  if (num >= 100000000) return (num / 100000000).toFixed(1) + '亿'
  if (num >= 10000) return (num / 10000).toFixed(1) + '万'
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.rank {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-weight: bold;
  color: white;
}

.rank-1 {
  background: linear-gradient(135deg, #ffd700, #ffb347);
}

.rank-2 {
  background: linear-gradient(135deg, #c0c0c0, #a0a0a0);
}

.rank-3 {
  background: linear-gradient(135deg, #cd7f32, #a0522d);
}

.model-name {
  font-family: monospace;
  font-size: 13px;
}

.user-rating-badge {
  font-size: 11px;
  color: #409eff;
  margin-top: 2px;
}

.detail-info {
  font-size: 12px;
  color: #909399;
  display: flex;
  gap: 10px;
}
</style>
