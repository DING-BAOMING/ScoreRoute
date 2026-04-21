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
      
      <el-tabs v-model="activeTab" style="margin-bottom: 20px">
        <el-tab-pane label="按格式类型" name="format">
          <div class="filter-bar">
            <el-select v-model="filterFormat" placeholder="筛选格式" clearable style="width: 150px" @change="filterChanged">
              <el-option v-for="fmt in formatOptions" :key="fmt" :label="fmt.toUpperCase()" :value="fmt" />
            </el-select>
            <el-select v-model="filterType" placeholder="筛选类型" clearable style="width: 150px" @change="filterChanged">
              <el-option v-for="t in typeOptions" :key="t" :label="getTypeName(t)" :value="t" />
            </el-select>
            <el-select v-model="filterFormatType" placeholder="格式+类型" clearable style="width: 200px" @change="filterChanged">
              <el-option v-for="ft in formatTypeOptions" :key="ft" :label="formatGroupName(ft)" :value="ft" />
            </el-select>
            <el-button @click="clearFilters">清除筛选</el-button>
          </div>
          <div v-for="(group, key) in filteredFormatTypeGroups" :key="key" class="model-group">
            <div class="group-header">
              <span class="group-title">{{ formatGroupName(key) }}</span>
              <span class="group-count">{{ group.length }} 个模型</span>
            </div>
            
            <el-table :data="group" v-loading="loading" stripe>
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
          </div>
        </el-tab-pane>
        <el-tab-pane label="按模型" name="model">
          <div class="filter-bar">
            <el-select v-model="filterModel" placeholder="筛选模型" clearable filterable style="width: 250px" @change="filterChanged">
              <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
            </el-select>
            <el-button @click="clearFilters">清除筛选</el-button>
          </div>
          <div v-for="(group, key) in filteredModelGroups" :key="key" class="model-group">
            <div class="group-header">
              <span class="group-title">{{ key }}</span>
              <span class="group-count">{{ group.length }} 个渠道</span>
            </div>
            
            <el-table :data="group" v-loading="loading" stripe>
              <el-table-column prop="rank" label="排名" width="80">
                <template #default="{ row }">
                  <span v-if="row.rank <= 3" :class="'rank rank-' + row.rank">{{ row.rank }}</span>
                  <span v-else>{{ row.rank }}</span>
                </template>
              </el-table-column>
              <el-table-column label="渠道" min-width="150">
                <template #default="{ row }">
                  <div class="model-name">{{ row.channel_name }}</div>
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
          </div>
        </el-tab-pane>
      </el-tabs>
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
const activeTab = ref('format')
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

const filterFormat = ref('')
const filterType = ref('')
const filterFormatType = ref('')
const filterModel = ref('')

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

function getTypeName(type) {
  const names = { 'chat': '文本', 'embedding': '嵌入', 'completions': '补全', 'images': '图像', 'audio': '音频', 'video': '视频' }
  return names[type] || type
}

function normalizeModelName(name) {
  if (!name) return 'unknown'
  let n = name.toLowerCase()
  const prefixes = ['minimaxai/', 'z-ai/', 'qwen/', 'meta/', 'mistralai/', 'microsoft/', 'anthropic/', 'cohere/', 'google/', 'openai/']
  for (const prefix of prefixes) {
    if (n.startsWith(prefix)) {
      n = n.substring(prefix.length)
      break
    }
  }
  if (n.startsWith('z/')) {
    n = n.substring(2)
  }
  return n || name
}

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
          type: sc.model_type,
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

function formatGroupName(key) {
  const typeNames = {
    'chat': '文本',
    'embedding': '嵌入',
    'completions': '补全',
    'images': '图像',
    'audio': '音频',
    'video': '视频'
  }
  const parts = key.split('_')
  if (parts.length >= 2) {
    const format = parts[0].toUpperCase()
    const type = typeNames[parts[1]] || parts[1]
    return `${format}-${type}`
  }
  return key || '未知'
}
const formatOptions = computed(() => {
  const formats = new Set()
  modelStats.value.forEach(m => formats.add(m.format))
  return Array.from(formats).sort()
})

const typeOptions = computed(() => {
  const types = new Set()
  modelStats.value.forEach(m => types.add(m.type))
  return Array.from(types).sort()
})

const formatTypeOptions = computed(() => {
  const combos = new Set()
  modelStats.value.forEach(m => combos.add(`${m.format}_${m.type}`))
  return Array.from(combos).sort()
})

const modelOptions = computed(() => {
  const models = new Set()
  modelStats.value.forEach(m => {
    models.add(normalizeModelName(m.model_name))
  })
  return Array.from(models).sort()
})

const filteredFormatTypeGroups = computed(() => {
  let filtered = modelStats.value
  
  if (filterFormat.value) {
    filtered = filtered.filter(m => m.format === filterFormat.value)
  }
  if (filterType.value) {
    filtered = filtered.filter(m => m.type === filterType.value)
  }
  if (filterFormatType.value) {
    const [fmt, type] = filterFormatType.value.split('_')
    filtered = filtered.filter(m => m.format === fmt && m.type === type)
  }
  
  const groups = {}
  filtered.forEach(model => {
    const key = `${model.format}_${model.type}`
    if (!groups[key]) groups[key] = []
    groups[key].push(model)
  })
  
  Object.keys(groups).forEach(key => {
    groups[key].sort((a, b) => b.score - a.score)
  })
  
  return groups
})

const filteredModelGroups = computed(() => {
  let filtered = modelStats.value
  
  if (filterModel.value) {
    const normalizedFilter = normalizeModelName(filterModel.value)
    filtered = filtered.filter(m => normalizeModelName(m.model_name) === normalizedFilter)
  }
  
  const groups = {}
  filtered.forEach(model => {
    const baseName = normalizeModelName(model.model_name)
    if (!groups[baseName]) groups[baseName] = []
    groups[baseName].push(model)
  })
  
  Object.keys(groups).forEach(key => {
    groups[key].sort((a, b) => b.score - a.score)
  })
  
  return groups
})

function filterChanged() {
  // Filters are reactive, computed properties handle the filtering
}

function clearFilters() {
  filterFormat.value = ''
  filterType.value = ''
  filterFormatType.value = ''
  filterModel.value = ''
}

const groupedModels = computed(() => {
  const groups = {}
  modelStats.value.forEach(model => {
    const format = model.format || 'unknown'
    const type = model.type || 'unknown'
    const key = `${format}_${type}`
    if (!groups[key]) {
      groups[key] = []
    }
    groups[key].push(model)
  })
  Object.keys(groups).forEach(key => {
    groups[key].sort((a, b) => b.score - a.score)
  })
  return groups
})

const modelGroupedModels = computed(() => {
  const groups = {}
  modelStats.value.forEach(model => {
    let modelBaseName = model.model_name || 'unknown'
    if (modelBaseName.includes('/')) {
      modelBaseName = modelBaseName.split('/')[1] || modelBaseName
    }
    if (!groups[modelBaseName]) {
      groups[modelBaseName] = []
    }
    groups[modelBaseName].push(model)
  })
  Object.keys(groups).forEach(key => {
    groups[key].sort((a, b) => b.score - a.score)
  })
  return groups
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

.model-group {
  margin-bottom: 30px;
}

.model-group:last-child {
  margin-bottom: 0;
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
  flex-wrap: wrap;
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px 8px 0 0;
  color: white;
}

.group-title {
  font-size: 16px;
  font-weight: bold;
}

.group-count {
  font-size: 13px;
  opacity: 0.9;
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
