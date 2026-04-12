<template>
  <div class="models">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>模型管理</span>
          <div class="header-actions">
            <el-select v-model="filterChannel" placeholder="筛选渠道" clearable style="width: 200px; margin-right: 10px" @change="loadData">
              <el-option v-for="ch in channels" :key="ch.id" :label="ch.name" :value="ch.id" />
            </el-select>
            <el-button type="warning" @click="showExchangeDialog">
              <el-icon><Coin /></el-icon>
              汇率设置
            </el-button>
            <el-button type="primary" @click="showDialog('create')">
              <el-icon><Plus /></el-icon>
              添加模型
            </el-button>
            <el-button type="success" @click="showBatchDialog">
              <el-icon><Upload /></el-icon>
              批量添加
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="channel_name" label="渠道" width="120">
          <template #default="{ row }">
            <el-tag type="info">{{ row.channel_name || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="模型名称" min-width="150" />
        <el-table-column label="限制" min-width="200">
          <template #default="{ row }">
            <div v-if="row.rate_limits && row.rate_limits !== '[]'" class="rate-limits">
              <span v-for="(rule, idx) in parseRateLimits(row.rate_limits)" :key="idx" class="rate-limit-tag">
                {{ rule.max_count }}/{{ rule.window }}
              </span>
            </div>
            <span v-else-if="row.total_token_limit > 0" class="token-limit">
              Token限制: {{ formatNumber(row.total_token_limit) }}
            </span>
            <span v-else class="no-limit">无限制</span>
          </template>
        </el-table-column>
        <el-table-column label="使用量" width="100">
          <template #default="{ row }">
            <div class="usage">
              <span>调用: {{ formatNumber(row.total_calls || row.call_count) }}</span>
              <span v-if="row.total_tokens > 0">Token: {{ formatNumber(row.total_tokens) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="费用" width="100">
          <template #default="{ row }">
            <span v-if="row.cost_per_token > 0">
              {{ row.cost_per_token }}/{{ row.currency }}
            </span>
            <span v-else class="no-limit">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240">
          <template #default="{ row }">
            <el-space :size="4">
              <el-button size="small" type="success" @click="testModel(row)">测试</el-button>
              <el-button size="small" type="primary" @click="showDialog('edit', row)">编辑</el-button>
              <el-button size="small" type="warning" @click="toggleEnabled(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" plain @click="deleteModel(row)">删除</el-button>
            </el-space>
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
        @current-change="loadData"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="700px" @closed="onDialogClosed">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="渠道" prop="channel_id">
          <el-select v-model="form.channel_id" placeholder="请选择渠道" style="width: 100%">
            <el-option v-for="ch in channels" :key="ch.id" :label="ch.name" :value="ch.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称" prop="name">
          <el-input v-model="form.name" placeholder="如: gpt-3.5-turbo" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
            <el-option label="Chat" value="chat" />
            <el-option label="Embedding" value="embedding" />
            <el-option label="Image" value="image" />
            <el-option label="Video" value="video" />
          </el-select>
        </el-form-item>
        
        <el-divider content-position="left">限制设置</el-divider>
        
        <el-form-item label="总Token限制">
          <el-input-number v-model="form.total_token_limit" :min="0" :step="1000000" placeholder="0表示无限制" style="width: 100%">
            <template #suffix>
              <span style="margin-right: 10px">Token</span>
            </template>
          </el-input-number>
          <div class="form-help">0表示无限制</div>
        </el-form-item>
        
        <el-form-item label="模型有效期">
          <el-date-picker
            v-model="form.expires_at"
            type="datetime"
            placeholder="不设置则永不过期"
            style="width: 100%"
            format="YYYY-MM-DD HH:mm"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
        
        <el-form-item label="调用频率限制">
          <div class="rate-limit-rules">
            <div v-for="(rule, idx) in rateLimitRules" :key="idx" class="rate-limit-rule">
              <el-select v-model="rule.type" style="width: 100px">
                <el-option label="调用次数" value="calls" />
                <el-option label="Token数" value="tokens" />
              </el-select>
              <el-input-number v-model="rule.max_count" :min="1" style="width: 120px" />
              <span style="width: 60px; text-align: center">次/</span>
              <el-select v-model="rule.window" style="width: 100px">
                <el-option label="分钟" value="minute" />
                <el-option label="小时" value="hour" />
                <el-option label="天" value="day" />
                <el-option label="周" value="week" />
                <el-option label="月" value="month" />
                <el-option label="年" value="year" />
              </el-select>
              <el-button type="danger" size="small" @click="removeRateLimitRule(idx)" :disabled="rateLimitRules.length <= 1">删除</el-button>
            </div>
            <el-button type="primary" size="small" @click="addRateLimitRule">添加限制规则</el-button>
            <div class="form-help">可添加多个限制规则，例如：每5小时最多1500次调用且每天最多3000次</div>
          </div>
        </el-form-item>
        
        <el-divider content-position="left">费用设置</el-divider>
        
        <el-form-item label="每Token费用">
          <el-input-number v-model="form.cost_per_token" :min="0" :precision="8" :step="0.00000001" style="width: 100%" />
          <div class="form-help">设置为0表示不计算费用</div>
        </el-form-item>
        
        <el-form-item label="货币单位">
          <el-select v-model="form.currency" style="width: 100%">
            <el-option label="人民币 (CNY)" value="CNY" />
            <el-option label="美元 (USD)" value="USD" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="info" @click="testModelDialog" :loading="testing">测试</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchDialogVisible" title="批量添加模型" width="600px">
      <el-form :model="batchForm" label-width="100px">
        <el-form-item label="渠道" required>
          <el-select v-model="batchForm.channel_id" placeholder="请选择渠道" style="width: 100%" @change="onChannelChange">
            <el-option v-for="ch in channels" :key="ch.id" :label="ch.name" :value="ch.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作">
          <el-button type="info" size="small" @click="fetchModels" :loading="fetchingModels">
            从API获取可用模型
          </el-button>
          <span v-if="availableModels.length > 0" style="margin-left: 10px">
            已获取 {{ availableModels.length }} 个模型
          </span>
        </el-form-item>
        <el-form-item label="模型名称">
          <el-select
            v-model="batchForm.model_names"
            multiple
            filterable
            allow-create
            placeholder="输入模型名称后回车添加，或从上方获取"
            style="width: 100%"
          >
            <el-option v-for="m in availableModels" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="batchForm.type" placeholder="请选择类型" style="width: 100%">
            <el-option label="Chat" value="chat" />
            <el-option label="Embedding" value="embedding" />
            <el-option label="Image" value="image" />
            <el-option label="Video" value="video" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitBatchForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="exchangeDialogVisible" title="汇率设置" width="500px">
      <el-form :model="exchangeForm" label-width="120px">
        <el-form-item label="USD兑换CNY汇率">
          <el-input-number v-model="exchangeForm.exchange_rate" :min="0.01" :precision="4" :step="0.01" style="width: 100%" />
          <div class="form-help">1美元可兑换的人民币数量</div>
        </el-form-item>
        <el-form-item label="默认货币">
          <el-select v-model="exchangeForm.currency" style="width: 100%">
            <el-option label="人民币 (CNY)" value="CNY" />
            <el-option label="美元 (USD)" value="USD" />
          </el-select>
          <div class="form-help">成本计算的默认货币单位</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exchangeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveExchangeConfig" :loading="exchangeSaving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { modelAPI, channelAPI, systemConfigAPI } from '../api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const channels = ref([])
const filterChannel = ref('')

const dialogVisible = ref(false)
const batchDialogVisible = ref(false)
const exchangeDialogVisible = ref(false)
const dialogType = ref('create')
const formRef = ref()
const testing = ref(false)
const submitting = ref(false)
const exchangeSaving = ref(false)

const form = reactive({
  id: null,
  channel_id: null,
  name: '',
  type: 'chat',
  rate_limits: '[]',
  total_token_limit: 0,
  expires_at: null,
  cost_per_token: 0,
  currency: 'CNY'
})

const rateLimitRules = ref([{
  type: 'calls',
  max_count: 1000,
  window: 'hour'
}])

const batchForm = reactive({
  channel_id: null,
  model_names: [],
  type: 'chat'
})

const availableModels = ref([])
const fetchingModels = ref(false)

const exchangeForm = reactive({
  exchange_rate: 7.25,
  currency: 'CNY'
})

const rules = {
  channel_id: [{ required: true, message: '请选择渠道', trigger: 'change' }],
  name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

const dialogTitle = computed(() => dialogType.value === 'create' ? '添加模型' : '编辑模型')

onMounted(() => {
  loadChannels()
  loadData()
  loadExchangeConfig()
})

async function loadExchangeConfig() {
  try {
    const res = await systemConfigAPI.get()
    if (res.code === 0 && res.data) {
      exchangeForm.exchange_rate = res.data.exchange_rate || 7.25
      exchangeForm.currency = res.data.currency || 'CNY'
    }
  } catch (e) {
    console.error('Failed to load exchange config', e)
  }
}

function showExchangeDialog() {
  exchangeDialogVisible.value = true
}

async function saveExchangeConfig() {
  exchangeSaving.value = true
  try {
    await systemConfigAPI.updateMultiple({
      exchange_rate: exchangeForm.exchange_rate,
      currency: exchangeForm.currency
    })
    ElMessage.success('汇率设置已保存')
    exchangeDialogVisible.value = false
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.message || '网络错误'))
  } finally {
    exchangeSaving.value = false
  }
}

async function loadChannels() {
  try {
    const res = await channelAPI.list({ page: 1, page_size: 100 })
    if (res.code === 0) {
      channels.value = res.data?.items || []
    }
  } catch (e) {
    console.error(e)
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await modelAPI.list({ page: page.value, page_size: pageSize.value })
    if (res.code === 0) {
      let items = res.data?.items || []
      if (filterChannel.value) {
        items = items.filter(m => m.channel_id === filterChannel.value)
      }
      list.value = items
      total.value = res.data?.total || 0
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function showDialog(type, row = null) {
  dialogType.value = type
  if (type === 'edit' && row) {
    Object.assign(form, {
      id: row.id,
      channel_id: row.channel_id,
      name: row.name,
      type: row.type || 'chat',
      rate_limits: row.rate_limits || '[]',
      total_token_limit: row.total_token_limit || 0,
      expires_at: row.expires_at,
      cost_per_token: row.cost_per_token || 0,
      currency: row.currency || 'CNY'
    })
    rateLimitRules.value = parseRateLimits(row.rate_limits)
    if (rateLimitRules.value.length === 0) {
      rateLimitRules.value = [{ type: 'calls', max_count: 1000, window: 'hour' }]
    }
  } else {
    Object.assign(form, {
      id: null,
      channel_id: null,
      name: '',
      type: 'chat',
      rate_limits: '[]',
      total_token_limit: 0,
      expires_at: null,
      cost_per_token: 0,
      currency: 'CNY'
    })
    rateLimitRules.value = [{ type: 'calls', max_count: 1000, window: 'hour' }]
  }
  dialogVisible.value = true
}

function parseRateLimits(rateLimitsStr) {
  try {
    return JSON.parse(rateLimitsStr || '[]')
  } catch {
    return []
  }
}

function formatNumber(num) {
  if (num >= 100000000) {
    return (num / 100000000).toFixed(1) + '亿'
  } else if (num >= 10000) {
    return (num / 10000).toFixed(1) + '万'
  }
  return num.toString()
}

function addRateLimitRule() {
  rateLimitRules.value.push({
    type: 'calls',
    max_count: 1000,
    window: 'hour'
  })
}

function removeRateLimitRule(idx) {
  rateLimitRules.value.splice(idx, 1)
}

function onDialogClosed() {
  rateLimitRules.value = [{ type: 'calls', max_count: 1000, window: 'hour' }]
}

async function testModelDialog() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  testing.value = true
  try {
    let modelId = form.id

    if (!modelId) {
      const res = await modelAPI.create(form)
      if (res.code === 0) {
        modelId = res.data?.id
        form.id = modelId
      } else {
        ElMessage.error(res.message || '保存模型失败')
        testing.value = false
        return
      }
    }

    const testRes = await modelAPI.test(modelId)
    if (testRes.code === 0) {
      const data = testRes.data
      if (data.success) {
        ElMessage.success(`测试成功! 延迟: ${data.latency_ms}ms`)
      } else {
        ElMessage.error(`测试失败: ${data.status_code} - ${JSON.stringify(data.response?.error || data.response)}`)
      }
    } else {
      ElMessage.error(testRes.message || '测试失败')
    }
  } catch (e) {
    ElMessage.error('测试失败: ' + (e.message || '网络错误'))
  } finally {
    testing.value = false
  }
}

function showBatchDialog() {
  batchForm.channel_id = null
  batchForm.model_names = []
  batchForm.type = 'chat'
  availableModels.value = []
  batchDialogVisible.value = true
}

function onChannelChange() {
  availableModels.value = []
}

async function fetchModels() {
  if (!batchForm.channel_id) {
    ElMessage.warning('请先选择渠道')
    return
  }
  fetchingModels.value = true
  try {
    const res = await channelAPI.fetchModels(batchForm.channel_id)
    if (res.code === 0) {
      availableModels.value = res.data || []
      ElMessage.success(`已获取 ${availableModels.value.length} 个可用模型`)
    } else {
      ElMessage.error(res.message || '获取失败')
    }
  } catch (e) {
    ElMessage.error('获取模型列表失败')
  } finally {
    fetchingModels.value = false
  }
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  form.rate_limits = JSON.stringify(rateLimitRules.value)
  
  submitting.value = true
  try {
    if (dialogType.value === 'create') {
      await modelAPI.create(form)
      ElMessage.success('添加成功')
    } else {
      await modelAPI.update(form.id, form)
      ElMessage.success('更新成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function submitBatchForm() {
  if (!batchForm.channel_id || batchForm.model_names.length === 0) {
    ElMessage.warning('请选择渠道并添加模型')
    return
  }

  submitting.value = true
  try {
    await modelAPI.batchCreate(batchForm)
    ElMessage.success('批量添加成功')
    batchDialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function toggleEnabled(row) {
  try {
    await modelAPI.setEnabled(row.id, row.enabled ? 0 : 1)
    ElMessage.success(row.enabled ? '已禁用' : '已启用')
    loadData()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function deleteModel(row) {
  await ElMessageBox.confirm(`确定要删除模型"${row.name}"吗？`, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
  try {
    await modelAPI.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

async function testModel(row) {
  ElMessage.info('正在测试模型...')
  try {
    const res = await modelAPI.test(row.id)
    if (res.code === 0) {
      const data = res.data
      if (data.success) {
        ElMessage.success(`测试成功! 延迟: ${data.latency_ms}ms`)
      } else {
        ElMessage.error(`测试失败: ${data.status_code} - ${JSON.stringify(data.response?.error || data.response)}`)
      }
    } else {
      ElMessage.error(res.message || '测试失败')
    }
  } catch (e) {
    ElMessage.error('测试失败')
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
}
.rate-limits {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.rate-limit-tag {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}
.token-limit {
  font-size: 12px;
  color: #909399;
}
.no-limit {
  font-size: 12px;
  color: #c0c4cc;
}
.usage {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  gap: 2px;
}
.rate-limit-rules {
  width: 100%;
}
.rate-limit-rule {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
