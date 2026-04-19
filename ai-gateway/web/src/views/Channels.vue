<template>
  <div class="channels">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>接入管理</span>
          <el-button type="primary" @click="showDialog('create')">
            <el-icon><Plus /></el-icon>
            添加渠道
          </el-button>
        </div>
      </template>
      
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="format" label="格式" width="100">
          <template #default="{ row }">
            <el-tag>{{ row.format }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="base_url" label="API地址" min-width="150" show-overflow-tooltip />
        <el-table-column label="限制" min-width="200">
          <template #default="{ row }">
            <div v-if="row.rate_limits && row.rate_limits !== '[]'" class="rate-limits">
              <span v-for="(rule, idx) in parseRateLimits(row.rate_limits)" :key="idx" class="rate-limit-tag">
                <span v-if="rule.type === 'billing'">{{ rule.max_count/100 }}{{ rule.currency }}/{{ rule.window }}</span>
                <span v-else>{{ rule.max_count }}/{{ rule.window }}</span>
              </span>
            </div>
            <span v-else-if="row.total_token_limit > 0" class="token-limit">
              Token限制: {{ formatNumber(row.total_token_limit) }}
            </span>
            <span v-else class="no-limit">无限制</span>
          </template>
        </el-table-column>
        <el-table-column label="使用量" width="120">
          <template #default="{ row }">
            <div class="usage">
              <span>调用: {{ formatNumber(row.total_calls || row.call_count) }}</span>
              <span v-if="row.total_tokens > 0">Token: {{ formatNumber(row.total_tokens) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <el-space :size="4">
              <el-button size="small" type="success" @click="testChannel(row)" :loading="row.testing">测试</el-button>
              <el-button size="small" type="primary" @click="showDialog('edit', row)">编辑</el-button>
              <el-button size="small" type="warning" @click="toggleEnabled(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" plain @click="deleteChannel(row)">删除</el-button>
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
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="渠道名称" />
        </el-form-item>
        <el-form-item label="格式" prop="format">
          <el-select v-model="form.format" placeholder="请选择格式" style="width: 100%">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Anthropic" value="anthropic" />
            <el-option label="Azure" value="azure" />
            <el-option label="Google" value="google" />
            <el-option label="智谱Zhipu" value="zhipu" />
          </el-select>
        </el-form-item>
        <el-form-item label="API地址" prop="base_url">
          <el-input v-model="form.base_url" placeholder="https://api.openai.com/v1" />
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input v-model="form.api_key" placeholder="sk-..." type="password" show-password />
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
        
        <el-form-item label="API有效期">
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
              <el-select v-model="rule.type" style="width: 120px" @change="onRuleTypeChange(rule)">
                <el-option label="调用次数" value="calls" />
                <el-option label="Token数" value="tokens" />
                <el-option label="周期计费" value="billing" />
              </el-select>
              <el-input-number v-model="rule.max_count" :min="1" style="width: 120px" />
              <span v-if="rule.type === 'billing'" style="width: 60px; text-align: center">元/</span>
              <span v-else style="width: 60px; text-align: center">次/</span>
              <el-select v-model="rule.window" style="width: 100px">
                <el-option label="分钟" value="minute" />
                <el-option label="小时" value="hour" />
                <el-option label="天" value="day" />
                <el-option label="周" value="week" />
                <el-option label="月" value="month" />
                <el-option label="季度" value="quarter" />
                <el-option label="年" value="year" />
              </el-select>
              <el-select v-if="rule.type === 'billing'" v-model="rule.currency" style="width: 90px">
                <el-option label="人民币" value="CNY" />
                <el-option label="美元" value="USD" />
              </el-select>
              <el-button type="danger" size="small" @click="removeRateLimitRule(idx)">删除</el-button>
            </div>
            <el-button type="primary" size="small" @click="addRateLimitRule">添加限制规则</el-button>
            <div class="form-help">可添加多个限制规则，例如：每5小时最多1500次调用且每天最多3000次；周期计费使用模型页面设置的汇率计算成本</div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="info" @click="testChannelDialog" :loading="testing">测试连接</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { channelAPI } from '../api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const dialogVisible = ref(false)
const dialogType = ref('create')
const formRef = ref()
const testing = ref(false)
const submitting = ref(false)

const form = reactive({
  name: '',
  format: 'openai',
  base_url: '',
  api_key: '',
  rate_limits: '[]',
  total_token_limit: 0,
  expires_at: null
})

const rateLimitRules = ref([])

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  format: [{ required: true, message: '请选择格式', trigger: 'change' }],
  base_url: [{ required: true, message: '请输入API地址', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入API Key', trigger: 'blur' }]
}

const defaultBaseUrls = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com',
  azure: '',
  google: 'https://generativelanguage.googleapis.com/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4'
}

watch(() => form.format, (newFormat) => {
  if (newFormat && defaultBaseUrls[newFormat] && !form.base_url) {
    form.base_url = defaultBaseUrls[newFormat]
  }
})

const dialogTitle = computed(() => dialogType.value === 'create' ? '添加渠道' : '编辑渠道')

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const res = await channelAPI.list({ page: page.value, page_size: pageSize.value })
    if (res.code === 0) {
      list.value = (res.data?.items || []).map(item => ({ ...item, testing: false }))
      total.value = res.data?.total || 0
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
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

function onRuleTypeChange(rule) {
  if (rule.type === 'billing' && !rule.currency) {
    rule.currency = 'CNY'
  }
}

function onDialogClosed() {
  rateLimitRules.value = [{
    type: 'calls',
    max_count: 1000,
    window: 'hour'
  }]
}

async function testChannel(row) {
  row.testing = true
  try {
    const res = await channelAPI.testCredentials({
      channel_id: row.id,
    })
    if (res.code === 0) {
      ElMessage.success('连接成功')
    } else {
      ElMessage.error(res.message || '测试失败')
    }
  } catch (e) {
    ElMessage.error(e.message || '测试失败')
  } finally {
    row.testing = false
  }
}

function showDialog(type, row = null) {
  dialogType.value = type
  if (type === 'edit' && row) {
    Object.assign(form, {
      id: row.id,
      name: row.name,
      format: row.format,
      rate_limits: row.rate_limits || '[]',
      total_token_limit: row.total_token_limit || 0,
      expires_at: row.expires_at
    })
    rateLimitRules.value = parseRateLimits(row.rate_limits)
    if (rateLimitRules.value.length === 0) {
      rateLimitRules.value = [{
        type: 'calls',
        max_count: 1000,
        window: 'hour'
      }]
    }
  } else {
    Object.assign(form, { name: '', format: 'openai', base_url: '', api_key: '', rate_limits: '[]', total_token_limit: 0, expires_at: null })
    rateLimitRules.value = [{
      type: 'calls',
      max_count: 1000,
      window: 'hour'
    }]
  }
  dialogVisible.value = true
}

async function testChannelDialog() {
  if (!form.base_url || !form.api_key) {
    ElMessage.warning('请先填写API地址和API Key')
    return
  }
  testing.value = true
  try {
    const res = await channelAPI.testCredentials({
      channel_id: row.id,
      base_url: form.base_url,
      api_key: form.api_key
    })
    if (res.code === 0) {
      ElMessage.success('连接成功')
    } else {
      ElMessage.error(res.message || '测试失败')
    }
  } catch (e) {
    ElMessage.error('测试连接失败: ' + (e.message || '网络错误'))
  } finally {
    testing.value = false
  }
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  form.rate_limits = JSON.stringify(rateLimitRules.value)
  
  submitting.value = true
  try {
    if (dialogType.value === 'create') {
      await channelAPI.create(form)
      ElMessage.success('添加成功')
    } else {
      await channelAPI.update(form.id, form)
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

async function toggleEnabled(row) {
  const action = row.enabled ? '禁用' : '启用'
  await ElMessageBox.confirm(`确定要${action}该渠道吗？`, '提示')
  try {
    await channelAPI.setEnabled(row.id, row.enabled ? 0 : 1)
    ElMessage.success(`${action}成功`)
    loadData()
  } catch (e) {
    ElMessage.error(`${action}失败`)
  }
}

async function deleteChannel(row) {
  await ElMessageBox.confirm(`确定要删除渠道"${row.name}"吗？删除后该渠道下的所有模型也将被删除！`, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
  try {
    await channelAPI.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
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
