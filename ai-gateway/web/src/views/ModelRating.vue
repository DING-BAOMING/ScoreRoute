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
        <el-table-column label="评分" width="120" sortable prop="score">
          <template #default="{ row }">
            <el-tag :type="getScoreType(row.score)" size="large">
              {{ row.score.toFixed(1) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="额外评分" width="120">
          <template #default="{ row }">
            <span v-if="row.extra_penalty || row.extra_reward" class="extra-rating">
              <span v-if="row.extra_penalty" class="penalty">-{{ row.extra_penalty }}</span>
              <span v-if="row.extra_reward" class="reward">+{{ row.extra_reward }}</span>
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column v-if="weights.cost_rating_weight > 0" label="成本" width="80">
          <template #default="{ row }">
            <span :class="row.cost_rating >= 70 ? 'rating-high' : row.cost_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.cost_rating.toFixed(0) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column v-if="weights.time_rating_weight > 0" label="时效" width="80">
          <template #default="{ row }">
            <span :class="row.time_rating >= 70 ? 'rating-high' : row.time_rating >= 40 ? 'rating-mid' : 'rating-low'">
              {{ row.time_rating.toFixed(0) }}
            </span>
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
import { ref, computed, onMounted } from 'vue'
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
  success_weight: 0.28,
  latency_weight: 0.21,
  reliability_weight: 0.21,
  user_rating_weight: 0.15,
  sample_rating_weight: 0.15,
  cost_rating_weight: 0,
  time_rating_weight: 0
})
const weightsForm = ref({...weights.value})

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const [statsRes, ratingsRes, sampleRatingsRes, extraRatingRes, costTimeRes, weightsRes] = await Promise.all([
      logAPI.modelStats(),
      userRatingAPI.listDeduplicated(),
      sampleAnalysisAPI.getRatingsMap(),
      extraRatingAPI.getModelScores(),
      modelRatingAPI.getCostTimeRatings(),
      modelRatingAPI.getWeights()
    ])
    
    if (weightsRes.code === 0 && weightsRes.data) {
      weights.value = {
        success_weight: weightsRes.data.success_weight || 0.28,
        latency_weight: weightsRes.data.latency_weight || 0.21,
        reliability_weight: weightsRes.data.reliability_weight || 0.21,
        user_rating_weight: weightsRes.data.user_rating_weight || 0.15,
        sample_rating_weight: weightsRes.data.sample_rating_weight || 0.15,
        cost_rating_weight: weightsRes.data.cost_rating_weight || 0,
        time_rating_weight: weightsRes.data.time_rating_weight || 0
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

      const penaltyMap = {}
      const rewardMap = {}
      if (extraRatingRes.code === 0 && extraRatingRes.data) {
        Object.entries(extraRatingRes.data).forEach(([key, value]) => {
          const normalizedKey = key.toLowerCase()
          penaltyMap[normalizedKey] = value.penalty || 0
          rewardMap[normalizedKey] = value.reward || 0
        })
      }
      extraPenaltyMap.value = penaltyMap
      extraRewardMap.value = rewardMap
      
      const stats = statsRes.data || []
      const scored = stats.map(s => {
        const userRating = getUserRatingForModel(s.model_name)
        const sampleRating = getSampleRatingForModel(s.model_name)
        const modelKey = normalizeModelKeyForExtra(s.channel_name, s.format, s.type, s.model_name)
        const extraPenalty = extraPenaltyMap.value[modelKey] || 0
        const extraReward = extraRewardMap.value[modelKey] || 0
        const costRating = getCostRatingForModel(modelKey)
        const timeRating = getTimeRatingForModel(modelKey)
        const score = calculateScore(s, userRating, sampleRating, extraPenalty, extraReward, costRating, timeRating)
        return { ...s, score, user_rating: userRating, sample_rating: sampleRating, extra_penalty: extraPenalty, extra_reward: extraReward, cost_rating: costRating, time_rating: timeRating }
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
  const w = weights.value
  const successWeight = w.success_weight
  const latencyWeight = w.latency_weight
  const reliabilityWeight = w.reliability_weight
  const userRatingWeight = w.user_rating_weight
  const sampleRatingWeight = w.sample_rating_weight
  const costRatingWeight = w.cost_rating_weight
  const timeRatingWeight = w.time_rating_weight

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
  const normalizedCostRating = costRating / 100
  const normalizedTimeRating = timeRating / 100

  const baseScore = successRate * successWeight + latencyScore * latencyWeight + reliabilityScore * reliabilityWeight + normalizedUserRating * userRatingWeight + normalizedSampleRating * sampleRatingWeight + normalizedCostRating * costRatingWeight + normalizedTimeRating * timeRatingWeight

  return baseScore * 100 + extraPenalty + extraReward
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
