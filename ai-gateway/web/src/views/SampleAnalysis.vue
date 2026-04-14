<template>
  <div class="sample-analysis">
    <el-tabs v-model="activeTab" class="analysis-tabs">
      <el-tab-pane label="样本展示" name="samples">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>样本列表</span>
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
            <el-table-column label="请求内容" min-width="150">
              <template #default="{ row }">
                <el-tooltip :content="row.request_content" placement="top" :show-after="300">
                  <div class="content-preview">{{ row.request_content }}</div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="响应内容" min-width="150">
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
      </el-tab-pane>

      <el-tab-pane label="样本评分" name="ratings">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>样本评分</span>
              <el-button type="primary" @click="loadRatings">
                <el-icon><Refresh /></el-icon>
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
                <li><b>总分</b>: 综合评分 (1-100)</li>
                <li><b>工具调用</b> (30%): 是否正确调用工具</li>
                <li><b>完整性</b> (25%): 是否完整回复请求</li>
                <li><b>上下文理解</b> (20%): 是否理解对话上下文</li>
                <li><b>错误处理</b> (15%): 错误处理能力</li>
                <li><b>回复质量</b> (10%): 回复质量评分</li>
              </ul>
            </template>
          </el-alert>

          <el-alert type="success" :closable="false" style="margin-bottom: 20px">
            <template #title>
              <span>智能重试策略 (完整提交 > 分块提交 > 摘要提交)</span>
            </template>
            <template #default>
              <div style="font-size: 12px; line-height: 1.6">
                <b>设计原则</b>: 能完整提交就完整提交 > 尽量分块提交 > 最后摘要提交<br/><br/>
                <b>重试流程</b>:<br/>
                <pre style="margin: 5px 0; padding: 8px; background: #f5f5f5; border-radius: 4px; font-size: 11px">
第一次: 完整提交 (保留开头内容 400/600/1000 字符)
    ↓ 失败?
    ├── 上下文过大问题
    │   ├── 第二次: 分块分析 (3部分评分取平均)
    │   │   ↓ 失败
    │   ├── 第三次: 尾部截断策略 (保留结尾)
    │   │   ↓ 失败
    │   └── 删除样本 (无法分析)
    │
    └── 非上下文问题
        ├── 第二次: 完整提交
        │   ↓ 失败
        ├── 第三次: 完整提交
        │   ↓ 失败
        └── 保留样本 (下次分析继续尝试)</pre>
                <b>错误分类</b>:<br/>
                <ul style="margin: 5px 0 0 0; padding-left: 20px">
                  <li><b>上下文错误</b>: "context length", "maximum context", "input tokens"</li>
                  <li><b>其他错误</b>: 网络超时、API错误、解析失败等</li>
                </ul>
              </div>
            </template>
          </el-alert>

          <el-table :data="ratings" v-loading="ratingsLoading" stripe>
            <el-table-column prop="model_key" label="模型名称" width="200" />
            <el-table-column prop="score" label="总分" width="100" sortable>
              <template #default="{ row }">
                <el-tag :type="getScoreType(row.score)" size="large">
                  {{ row.score }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="tool_calling_score" label="工具调用" width="100" sortable>
              <template #default="{ row }">
                {{ row.tool_calling_score }}
              </template>
            </el-table-column>
            <el-table-column prop="completeness_score" label="完整性" width="100" sortable>
              <template #default="{ row }">
                {{ row.completeness_score }}
              </template>
            </el-table-column>
            <el-table-column prop="context_understanding_score" label="上下文理解" width="120" sortable>
              <template #default="{ row }">
                {{ row.context_understanding_score }}
              </template>
            </el-table-column>
            <el-table-column prop="error_handling_score" label="错误处理" width="100" sortable>
              <template #default="{ row }">
                {{ row.error_handling_score }}
              </template>
            </el-table-column>
            <el-table-column prop="response_quality_score" label="回复质量" width="100" sortable>
              <template #default="{ row }">
                {{ row.response_quality_score }}
              </template>
            </el-table-column>
            <el-table-column label="分析时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.analyzed_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="warning" @click="editRating(row)">修改评分</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="分析日志" name="logs">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>样本分析日志</span>
              <el-button type="primary" @click="loadAnalysisLogs">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <el-row :gutter="20" style="margin-bottom: 20px">
            <el-col :span="6">
              <el-statistic title="总分析数" :value="logStats.total" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="成功" :value="logStats.success_count" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="失败" :value="logStats.failed_count" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="平均分" :value="logStats.avg_score" suffix="分" />
            </el-col>
          </el-row>

          <el-table :data="analysisLogs" v-loading="logsLoading" stripe>
            <el-table-column prop="model_key" label="模型名称" width="200" />
            <el-table-column prop="score" label="得分" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.success" type="success">{{ row.score }}</el-tag>
                <el-tag v-else type="danger">失败</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="分析时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.analysis_time) }}
              </template>
            </el-table-column>
            <el-table-column label="删除时间" width="180">
              <template #default="{ row }">
                {{ row.delete_time ? formatTime(row.delete_time) : '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="error_message" label="错误信息" min-width="200">
              <template #default="{ row }">
                <span v-if="row.error_message" class="error-text">{{ row.error_message }}</span>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="LLM设置" name="settings">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>样本分析LLM设置</span>
              <el-button type="primary" @click="loadConfig">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <el-form :model="configForm" label-width="120px" style="max-width: 600px">
            <el-form-item label="API格式">
              <el-select v-model="configForm.format" placeholder="选择API格式">
                <el-option label="OpenAI" value="openai" />
                <el-option label="Anthropic" value="anthropic" />
                <el-option label="Google" value="google" />
                <el-option label="Zhipu" value="zhipu" />
              </el-select>
            </el-form-item>

            <el-form-item label="API地址">
              <el-input v-model="configForm.base_url" placeholder="https://api.openai.com/v1" />
            </el-form-item>

            <el-form-item label="API Key">
              <el-input v-model="configForm.api_key" type="password" show-password placeholder="sk-..." />
            </el-form-item>

            <el-form-item label="模型名称">
              <el-input v-model="configForm.model_name" placeholder="gpt-4, claude-3-opus, etc." />
            </el-form-item>

            <el-form-item label="启用">
              <el-switch v-model="configForm.enabled" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleTestConfig" :loading="testing">测试连接</el-button>
              <el-button type="success" @click="handleSaveConfig">保存设置</el-button>
              <el-button type="warning" @click="handleRunAnalysis" :loading="running">运行分析</el-button>
            </el-form-item>
          </el-form>

          <el-alert type="info" :closable="false" style="margin-top: 20px">
            <template #title>
              <span>分析逻辑说明</span>
            </template>
            <template #default>
              <ul style="margin: 5px 0 0 0; padding-left: 20px">
                <li><b>分析频率</b>: 每2小时自动分析最多20个样本</li>
                <li><b>分析顺序</b>: 按剩余时间升序（即将过期的优先分析）</li>
                <li><b>分析后处理</b>: 分析完成后自动删除样本</li>
                <li><b>评分有效期</b>: 评分保存7天，过期后自动清理</li>
                <li><b>评分同步</b>: 样本评分自动同步到模型评分页面</li>
              </ul>
            </template>
          </el-alert>
        </el-card>
      </el-tab-pane>
    </el-tabs>

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
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ratingEditVisible" title="修改评分" width="500px">
      <el-form :model="ratingEditForm" label-width="100px">
        <el-form-item label="模型名称">
          <el-input v-model="ratingEditForm.model_key" disabled />
        </el-form-item>
        <el-form-item label="评分">
          <el-slider v-model="ratingEditForm.score" :min="1" :max="100" show-input />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ratingEditVisible = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateRating">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { sampleAPI, sampleAnalysisAPI } from '../api'

const activeTab = ref('samples')
const loading = ref(false)
const samples = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const detailVisible = ref(false)
const currentSample = ref(null)

const ratingsLoading = ref(false)
const ratings = ref([])

const logsLoading = ref(false)
const analysisLogs = ref([])
const logStats = reactive({
  total: 0,
  success_count: 0,
  failed_count: 0,
  avg_score: 0
})

const testing = ref(false)
const running = ref(false)
const configForm = reactive({
  format: 'openai',
  base_url: '',
  api_key: '',
  model_name: 'gpt-4',
  enabled: 1
})

const ratingEditVisible = ref(false)
const ratingEditForm = reactive({
  model_key: '',
  score: 50
})

const stats = reactive({
  total: 0,
  models: 0,
  avgTokens: 0,
  expired: 0
})

onMounted(() => {
  loadSamples()
  loadStats()
  loadConfig()
  loadRatings()
  loadAnalysisLogs()
  loadLogStats()
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

async function loadConfig() {
  try {
    const res = await sampleAnalysisAPI.getConfig()
    if (res.code === 0 && res.data) {
      configForm.format = res.data.format || 'openai'
      configForm.base_url = res.data.base_url || ''
      configForm.api_key = res.data.api_key || ''
      configForm.model_name = res.data.model_name || 'gpt-4'
      configForm.enabled = res.data.enabled || 0
    }
  } catch (e) {
    console.error('加载配置失败', e)
  }
}

async function handleTestConfig() {
  testing.value = true
  try {
    const res = await sampleAnalysisAPI.testConfig({
      format: configForm.format,
      base_url: configForm.base_url,
      api_key: configForm.api_key,
      model_name: configForm.model_name
    })
    if (res.code === 0) {
      ElMessage.success(res.message)
    } else {
      ElMessage.warning(res.message)
    }
  } catch (e) {
    ElMessage.error('测试连接失败')
  } finally {
    testing.value = false
  }
}

async function handleSaveConfig() {
  try {
    const res = await sampleAnalysisAPI.saveConfig({
      format: configForm.format,
      base_url: configForm.base_url,
      api_key: configForm.api_key,
      model_name: configForm.model_name,
      enabled: configForm.enabled
    })
    if (res.code === 0) {
      ElMessage.success(res.message)
    } else {
      ElMessage.error(res.message)
    }
  } catch (e) {
    ElMessage.error('保存配置失败')
  }
}

async function handleRunAnalysis() {
  running.value = true
  try {
    const res = await sampleAnalysisAPI.runAnalysis()
    if (res.code === 0) {
      ElMessage.success(res.message + ', 已分析 ' + res.data.analyzed + ' 个样本')
      loadAnalysisLogs()
      loadLogStats()
      loadRatings()
      loadSamples()
      loadStats()
    } else {
      ElMessage.error(res.message)
    }
  } catch (e) {
    ElMessage.error('运行分析失败')
  } finally {
    running.value = false
  }
}

async function loadRatings() {
  ratingsLoading.value = true
  try {
    const res = await sampleAnalysisAPI.getRatings()
    if (res.code === 0) {
      ratings.value = res.data || []
    }
  } catch (e) {
    ElMessage.error('加载评分失败')
  } finally {
    ratingsLoading.value = false
  }
}

function editRating(row) {
  ratingEditForm.model_key = row.model_key
  ratingEditForm.score = row.score
  ratingEditVisible.value = true
}

async function handleUpdateRating() {
  try {
    const res = await sampleAnalysisAPI.updateRating({
      model_key: ratingEditForm.model_key,
      score: ratingEditForm.score
    })
    if (res.code === 0) {
      ElMessage.success(res.message)
      ratingEditVisible.value = false
      loadRatings()
    } else {
      ElMessage.error(res.message)
    }
  } catch (e) {
    ElMessage.error('更新评分失败')
  }
}

async function loadAnalysisLogs() {
  logsLoading.value = true
  try {
    const res = await sampleAnalysisAPI.getLogs()
    if (res.code === 0) {
      analysisLogs.value = res.data || []
    }
  } catch (e) {
    ElMessage.error('加载日志失败')
  } finally {
    logsLoading.value = false
  }
}

async function loadLogStats() {
  try {
    const res = await sampleAnalysisAPI.getLogStats()
    if (res.code === 0) {
      logStats.total = res.data?.total || 0
      logStats.success_count = res.data?.success_count || 0
      logStats.failed_count = res.data?.failed_count || 0
      logStats.avg_score = Math.round(res.data?.avg_score || 0)
    }
  } catch (e) {
    console.error('加载日志统计失败', e)
  }
}

function viewDetail(row) {
  currentSample.value = row
  detailVisible.value = true
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

function getScoreType(score) {
  if (score >= 80) return 'success'
  if (score >= 60) return 'warning'
  return 'danger'
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
  max-width: 150px;
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

.error-text {
  color: #f56c6c;
  font-size: 12px;
}

.analysis-tabs :deep(.el-tabs__content) {
  overflow: visible;
}
</style>
