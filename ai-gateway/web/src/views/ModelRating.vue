<template>
  <div class="model-rating">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>模型评分</span>
          <div>
            <el-button type="warning" @click="showWeightDialog = true">
              <el-icon><Setting /></el-icon>
              设置权重
            </el-button>
            <el-button type="primary" @click="loadData">
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-alert type="info" :closable="false" style="margin-bottom: 20px">
        <template #title>
          <span>评分说明（百分制，可配置）</span>
        </template>
        <template #default>
          <ul style="margin: 5px 0 0 0; padding-left: 20px">
            <li><b>成功率</b> ({{ (weights.success_weight * 100).toFixed(0) }}%): 成功请求占总请求的比例</li>
            <li><b>延迟分数</b> ({{ (weights.latency_weight * 100).toFixed(0) }}%): 基于平均延迟计算，延迟越低分数越高</li>
            <li><b>稳定性</b> ({{ (weights.reliability_weight * 100).toFixed(0) }}%): 基于样本量计算，样本越多评分越可靠</li>
            <li><b>用户评分</b> ({{ (weights.user_rating_weight * 100).toFixed(0) }}%): 用户对模型的评分（1-100标准化）</li>
            <li><b>样本分析</b> ({{ (weights.sample_rating_weight * 100).toFixed(0) }}%): 基于样本分析评分，评估工具调用、完整性等</li>
            <li v-if="weights.cost_rating_weight > 0"><b>成本评分</b> ({{ (weights.cost_rating_weight * 100).toFixed(0) }}%): 基于API价格计算，免费/低价模型得分更高</li>
            <li v-if="weights.time_rating_weight > 0"><b>时效评分</b> ({{ (weights.time_rating_weight * 100).toFixed(0) }}%): 基于API有效期计算，即将过期的模型得分更高</li>
            <li><b>额外评分</b>: 惩罚/奖励分数实时调整</li>
            <li><b>权重配置</b>: 点击右上角"设置权重"按钮调整各评分权重</li>
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
          </template>
        </el-table-column>
        <el-table-column label="成功率" width="130" sortable prop="success_rate_percent">
          <template #default="{ row }">
            <span :class="row.success_rate >= 95 ? 'rating-high' : row.success_rate >= 80 ? 'rating-mid' : 'rating-low'">
              {{ row.success_rate_percent.toFixed(0) }}/{{ row.success_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="延迟分数" width="130" sortable prop="latency_score_percent">
          <template #default="{ row }">
            <span :class="row.latency_score_percent >= 70 ? 'rating-high' : row.latency_score_percent >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.latency_score_percent.toFixed(0) }}/{{ row.latency_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="稳定性" width="130" sortable prop="reliability_score_percent">
          <template #default="{ row }">
            <span :class="row.reliability_score_percent >= 70 ? 'rating-high' : row.reliability_score_percent >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.reliability_score_percent.toFixed(0) }}/{{ row.reliability_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="用户评分" width="130" sortable prop="user_rating">
          <template #default="{ row }">
            <span :class="row.user_rating >= 70 ? 'rating-high' : row.user_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.user_rating }}/{{ row.user_rating_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="样本评分" width="130" sortable prop="sample_rating">
          <template #default="{ row }">
            <span v-if="row.sample_rating > 0" :class="row.sample_rating >= 70 ? 'rating-high' : row.sample_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.sample_rating }}/{{ row.sample_rating_weighted.toFixed(1) }}
            </span>
            <span v-else class="rating-none">-/-</span>
          </template>
        </el-table-column>
        <el-table-column label="成本评分" width="130" sortable prop="cost_rating">
          <template #default="{ row }">
            <span :class="row.cost_rating >= 70 ? 'rating-high' : row.cost_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.cost_rating.toFixed(0) }}/{{ row.cost_rating_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="时效评分" width="130" sortable prop="time_rating">
          <template #default="{ row }">
            <span :class="row.time_rating >= 70 ? 'rating-high' : row.time_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.time_rating.toFixed(0) }}/{{ row.time_rating_weighted.toFixed(1) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="额外评分" width="100">
          <template #default="{ row }">
            <span v-if="row.extra_penalty || row.extra_reward" class="extra-rating">
              <span v-if="row.extra_penalty" class="penalty">{{ row.extra_penalty }}</span>
              <span v-if="row.extra_reward" class="reward">{{ row.extra_reward > 0 ? '+' : '' }}{{ row.extra_reward }}</span>
            </span>
            <span v-else class="rating-none">0</span>
          </template>
        </el-table-column>
        <el-table-column label="综合评分" width="120" sortable prop="score">
          <template #default="{ row }">
            <el-tag :type="getScoreType(row.score)" size="large">
              {{ row.score.toFixed(1) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="详情" width="150">
          <template #default="{ row }">
            <div class="detail-info">
              <span>调用: {{ formatNumber(row.total_calls) }}</span>
              <span>Token: {{ formatNumber(row.total_tokens) }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showWeightDialog" title="评分权重设置" width="700px">
      <el-form :model="weightsForm" label-width="120px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="成功率">
              <el-slider v-model="weightsForm.success_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
            <el-form-item label="延迟">
              <el-slider v-model="weightsForm.latency_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
            <el-form-item label="稳定性">
              <el-slider v-model="weightsForm.reliability_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户评分">
              <el-slider v-model="weightsForm.user_rating_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
            <el-form-item label="样本分析">
              <el-slider v-model="weightsForm.sample_rating_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
            <el-form-item label="成本评分">
              <el-slider v-model="weightsForm.cost_rating_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
            <el-form-item label="时效评分">
              <el-slider v-model="weightsForm.time_rating_weight" :min="0" :max="1" :step="0.01" show-input />
            </el-form-item>
          </el-col>
        </el-row>
        <div class="weight-summary">
          权重总和: <span :class="totalWeightValid ? 'weight-valid' : 'weight-invalid'">{{ totalWeight.toFixed(2) }}</span>
          <span v-if="totalWeightValid" class="weight-ok"> ✓</span>
          <span v-else class="weight-error"> 权重总和必须在0.1到1之间</span>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="showWeightDialog = false">取消</el-button>
        <el-button type="primary" @click="saveWeights" :loading="savingWeights">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { logAPI, userRatingAPI, sampleAnalysisAPI, extraRatingAPI, modelRatingAPI } from '../api'

const loading = ref(false)
const savingWeights = ref(false)
const showWeightDialog = ref(false)
const modelStats = ref([])
const userRatings = ref({})
const sampleRatings = ref({})
const extraPenaltyMap = ref({})
const extraRewardMap = ref({})
const costTimeRatings = ref({})
const weights = ref({
  success_weight: 0.15,
  latency_weight: 0.1,
  reliability_weight: 0.1,
  user_rating_weight: 0.15,
  sample_rating_weight: 0.25,
  cost_rating_weight: 0.15,
  time_rating_weight: 0.1
})
const weightsForm = ref({...weights.value})

let refreshInterval = null

onMounted(() => {
  loadData()
  refreshInterval = setInterval(() => {
    loadData()
  }, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})

async function loadData() {
  loading.value = true
  try {
    const [scoresRes, weightsRes, statsRes, costTimeRes] = await Promise.all([
      modelRatingAPI.getAllScores(),
      modelRatingAPI.getWeights(),
      logAPI.modelStats(),
      modelRatingAPI.getCostTimeRatings()
    ])
    
    if (weightsRes.code === 0 && weightsRes.data) {
      weights.value = {
        success_weight: weightsRes.data.success_weight || 0.15,
        latency_weight: weightsRes.data.latency_weight || 0.1,
        reliability_weight: weightsRes.data.reliability_weight || 0.1,
        user_rating_weight: weightsRes.data.user_rating_weight || 0.15,
        sample_rating_weight: weightsRes.data.sample_rating_weight || 0.25,
        cost_rating_weight: weightsRes.data.cost_rating_weight || 0.15,
        time_rating_weight: weightsRes.data.time_rating_weight || 0.1
      }
      weightsForm.value = {...weights.value}
    }

    if (costTimeRes.code === 0 && Array.isArray(costTimeRes.data)) {
      const ctMap = {}
      costTimeRes.data.forEach(item => {
        ctMap[item.model_key.toLowerCase()] = item
      })
      costTimeRatings.value = ctMap
    }

    const statsMap = {}
    if (statsRes.code === 0) {
      ;(statsRes.data || []).forEach(s => {
        const key = `${s.channel_name}_${s.format}_${s.type}_${s.model_name}`.toLowerCase()
        statsMap[key] = s
      })
    }

    if (scoresRes.code === 0 && Array.isArray(scoresRes.data)) {
      const w = weights.value
      modelStats.value = scoresRes.data.map(sc => {
        const modelKey = sc.model_key.toLowerCase()
        const stats = statsMap[modelKey]
        const costTime = costTimeRatings.value[modelKey] || {}
        
        const successRatePercent = sc.success_rate
        const latencyScorePercent = sc.latency
        const reliabilityScorePercent = sc.reliability
        const userRatingPercent = sc.user_rating
        const sampleRatingPercent = sc.sample_rating
        const costRatingPercent = costTime.cost_rating || 50
        const timeRatingPercent = costTime.time_rating || 50

        const successWeighted = successRatePercent * w.success_weight
        const latencyWeighted = latencyScorePercent * w.latency_weight
        const reliabilityWeighted = reliabilityScorePercent * w.reliability_weight
        const userRatingWeighted = userRatingPercent * w.user_rating_weight
        const sampleRatingWeighted = sampleRatingPercent * w.sample_rating_weight
        const costRatingWeighted = costRatingPercent * w.cost_rating_weight
        const timeRatingWeighted = timeRatingPercent * w.time_rating_weight

        return {
          ...sc,
          success_rate_percent: successRatePercent,
          latency_score_percent: latencyScorePercent,
          reliability_score_percent: reliabilityScorePercent,
          user_rating: userRatingPercent,
          sample_rating: sampleRatingPercent,
          cost_rating: costRatingPercent,
          time_rating: timeRatingPercent,
          success_weighted: successWeighted,
          latency_weighted: latencyWeighted,
          reliability_weighted: reliabilityWeighted,
          user_rating_weighted: userRatingWeighted,
          sample_rating_weighted: sampleRatingWeighted,
          cost_rating_weighted: costRatingWeighted,
          time_rating_weighted: timeRatingWeighted,
          extra_penalty: sc.penalty || 0,
          extra_reward: sc.reward || 0,
          total_calls: stats ? stats.total_calls : 0,
          total_tokens: stats ? stats.total_tokens : 0
        }
      })
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

function getSampleRatingForModel(modelKey) {
  if (!modelKey) return 0
  const rating = sampleRatings.value[modelKey.toLowerCase()]
  return rating || 0
}

function getCostRatingForModel(modelKey) {
  const rating = costTimeRatings.value[modelKey.toLowerCase()]
  return rating ? rating.cost_rating : 50
}

function getTimeRatingForModel(modelKey) {
  const rating = costTimeRatings.value[modelKey.toLowerCase()]
  return rating ? rating.time_rating : 50
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

function normalizeModelKeyForExtra(channelName, format, modelType, modelName) {
  return `${channelName}_${format}_${modelType}_${modelName}`.toLowerCase()
}

function calculateScore(stat, userRating, sampleRating, extraPenalty = 0, extraReward = 0, costRating = 50, timeRating = 50) {
  const { score } = calculateScoreDetailed(stat, userRating, sampleRating, extraPenalty, extraReward, costRating, timeRating)
  return score
}

function calculateScoreDetailed(stat, userRating, sampleRating, extraPenalty = 0, extraReward = 0, costRating = 50, timeRating = 50) {
  const w = weights.value
  const successWeight = w.success_weight
  const latencyWeight = w.latency_weight
  const reliabilityWeight = w.reliability_weight
  const userRatingWeight = w.user_rating_weight
  const sampleRatingWeight = w.sample_rating_weight
  const costRatingWeight = w.cost_rating_weight
  const timeRatingWeight = w.time_rating_weight

  const successRatePercent = stat.success_rate

  const maxLatencyThreshold = 30000
  const latencyScorePercent = Math.max(0, 1 - (stat.avg_latency / maxLatencyThreshold)) * 100

  let reliabilityScorePercent = 85
  if (stat.total_calls >= 30) {
    reliabilityScorePercent = 100
  } else if (stat.total_calls >= 10) {
    reliabilityScorePercent = 80 + (stat.total_calls - 10) / 20 * 20
  } else if (stat.total_calls >= 5) {
    reliabilityScorePercent = 50 + (stat.total_calls - 5) / 5 * 30
  } else if (stat.total_calls > 0) {
    reliabilityScorePercent = 50
  }

  const userRatingPercent = userRating
  const sampleRatingPercent = sampleRating
  const costRatingPercent = costRating
  const timeRatingPercent = timeRating

  const successWeighted = successRatePercent * successWeight
  const latencyWeighted = latencyScorePercent * latencyWeight
  const reliabilityWeighted = reliabilityScorePercent * reliabilityWeight
  const userRatingWeighted = userRatingPercent * userRatingWeight
  const sampleRatingWeighted = sampleRatingPercent * sampleRatingWeight
  const costRatingWeighted = costRatingPercent * costRatingWeight
  const timeRatingWeighted = timeRatingPercent * timeRatingWeight

  const baseScore = successWeighted + latencyWeighted + reliabilityWeighted + userRatingWeighted + sampleRatingWeighted + costRatingWeighted + timeRatingWeighted

  const score = baseScore + extraPenalty + extraReward
  
  return { 
    score, 
    successRatePercent, latencyScorePercent, reliabilityScorePercent,
    userRatingPercent, sampleRatingPercent, costRatingPercent, timeRatingPercent,
    successWeighted, latencyWeighted, reliabilityWeighted,
    userRatingWeighted, sampleRatingWeighted, costRatingWeighted, timeRatingWeighted
  }
}

const displayList = computed(() => {
  return modelStats.value
})

const totalWeight = computed(() => {
  return weightsForm.value.success_weight + 
         weightsForm.value.latency_weight + 
         weightsForm.value.reliability_weight + 
         weightsForm.value.user_rating_weight + 
         weightsForm.value.sample_rating_weight + 
         weightsForm.value.cost_rating_weight + 
         weightsForm.value.time_rating_weight
})

const totalWeightValid = computed(() => {
  return totalWeight.value >= 0.1 && totalWeight.value <= 1.0
})

async function saveWeights() {
  if (!totalWeightValid.value) {
    ElMessage.error('权重总和必须在0.1到1之间')
    return
  }
  savingWeights.value = true
  try {
    const res = await modelRatingAPI.updateWeights(weightsForm.value)
    if (res.code === 0) {
      weights.value = {...weightsForm.value}
      showWeightDialog.value = false
      ElMessage.success('保存成功')
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('保存权重失败')
  } finally {
    savingWeights.value = false
  }
}

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

.extra-rating {
  display: flex;
  gap: 5px;
  font-size: 12px;
}

.extra-rating .penalty {
  color: #f56c6c;
}

.extra-rating .reward {
  color: #67c23a;
}

.rating-high {
  color: #67c23a;
  font-weight: bold;
}

.rating-mid {
  color: #e6a23c;
}

.rating-low {
  color: #f56c6c;
}

.rating-none {
  color: #909399;
}

.weight-summary {
  text-align: center;
  font-size: 16px;
  margin-top: 20px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
}

.weight-valid {
  color: #67c23a;
  font-weight: bold;
}

.weight-invalid {
  color: #f56c6c;
  font-weight: bold;
}

.weight-ok {
  color: #67c23a;
  font-weight: bold;
}

.weight-error {
  color: #f56c6c;
  font-size: 14px;
}
</style>
