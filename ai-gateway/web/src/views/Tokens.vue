<template>
  <div class="tokens">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>接出管理</span>
          <el-button type="primary" @click="showDialog">
            <el-icon><Plus /></el-icon>
            创建API
          </el-button>
        </div>
      </template>
      
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="key" label="API Key" min-width="200">
          <template #default="{ row }">
            <code>{{ row.key }}</code>
            <el-button size="small" text @click="copyKey(row.key)">
              <el-icon><CopyDocument /></el-icon>
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="format" label="格式" width="100">
          <template #default="{ row }">
            <el-tag>{{ row.format }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag type="success">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="模型" width="180">
          <template #default="{ row }">
            {{ row.model_name === '__POLL_ALL__' ? '轮询所有' : (row.model_name === '__AUTO__' ? '自动选择' : (row.model_name || '轮询所有')) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250">
          <template #default="{ row }">
            <el-button size="small" @click="editToken(row)">编辑</el-button>
            <el-button size="small" @click="toggleEnabled(row)">
              {{ row.enabled ? '禁用' : '启用' }}
            </el-button>
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
        @current-change="(p) => { page.value = p; loadData() }"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑API' : '创建API'" width="500px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="API名称" />
        </el-form-item>
        <el-form-item label="格式" prop="format">
          <el-select v-model="form.format" placeholder="请选择格式" style="width: 100%">
            <el-option label="OpenAI" value="openai" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
            <el-option label="Chat" value="chat" />
            <el-option label="Embedding" value="embedding" />
            <el-option label="Image" value="image" />
            <el-option label="Video" value="video" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称" prop="model_name">
          <el-select v-model="form.model_name" placeholder="留空则轮询所有模型" style="width: 100%">
            <el-option label="轮询所有模型（自动分配）" value="__POLL_ALL__" />
            <el-option label="自动选择（同格式同类型轮询）" value="__AUTO__" />
            <el-option v-for="m in models" :key="m.id" :label="m.name" :value="m.name" />
          </el-select>
        </el-form-item>

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
              <el-select v-model="rule.type" style="width: 120px">
                <el-option label="调用次数" value="calls" />
                <el-option label="Token数" value="tokens" />
              </el-select>
              <el-input-number v-model="rule.max_count" :min="1" style="width: 120px" />
              <span v-if="rule.type === 'tokens'" style="width: 60px; text-align: center">Token/</span>
              <span v-else style="width: 60px; text-align: center">次/</span>
              <el-select v-model="rule.window" style="width: 100px">
                <el-option label="分钟" value="minute" />
                <el-option label="小时" value="hour" />
                <el-option label="天" value="day" />
                <el-option label="周" value="week" />
                <el-option label="月" value="month" />
                <el-option label="年" value="year" />
              </el-select>
              <el-button type="danger" size="small" @click="removeRateLimitRule(idx)">删除</el-button>
            </div>
            <el-button type="primary" size="small" @click="addRateLimitRule">添加限制规则</el-button>
            <div class="form-help">可添加多个限制规则，例如：每小时最多1000次调用且每天最多5000次</div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">{{ isEdit ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resultDialogVisible" title="API创建成功" width="600px">
      <el-alert type="success" :closable="false" style="margin-bottom: 20px">
        请妥善保存您的API Key，关闭后将无法再次查看完整Key
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="API地址">
          <el-input :value="apiUrl" readonly />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input :value="createdKey" readonly />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="copyKey(createdKey); resultDialogVisible = false">复制并关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { tokenAPI, modelAPI, modelRatingAPI } from '../api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const models = ref([])

const dialogVisible = ref(false)
const resultDialogVisible = ref(false)
const formRef = ref()
const createdKey = ref('')
const apiUrl = computed(() => window.location.origin + '/v1')
const isEdit = ref(false)
const editingId = ref(null)

const form = reactive({
  name: '',
  format: 'openai',
  type: 'chat',
  model_name: '',
  total_token_limit: 0,
  expires_at: null
})

const rateLimitRules = ref([])

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  format: [{ required: true, message: '请选择格式', trigger: 'change' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

onMounted(() => {
  loadData()
  loadModels()
})

function normalizeModelName(modelName) {
  if (!modelName) return ''
  let n = modelName.toLowerCase()
  
  // Strip common provider prefixes
  const prefixes = ['minimaxai/', 'z-ai/', 'qwen/', 'meta/', 'mistralai/', 'microsoft/', 'anthropic/', 'cohere/', 'google/', 'openai/', 'azure/', 'aws/', 'alibaba/', 'baidu/', 'tencent/']
  for (const prefix of prefixes) {
    if (n.startsWith(prefix)) {
      n = n.substring(prefix.length)
      break
    }
  }
  
  // Handle minimax variations: minimaxai/minimax-m2.7 -> minimax-m2.7
  if (n.startsWith('minimax-') || n.startsWith('minimax')) {
    n = n.replace(/^minimax-?/, 'minimax-')
  }
  
  // If contains '/', take the last part
  if (n.includes('/')) {
    const parts = n.split('/')
    n = parts[parts.length - 1]
  }
  
  // Capitalize first letter for display
  if (n.length > 0) {
    n = n.charAt(0).toUpperCase() + n.substring(1)
  }
  
  return n
}

async function loadModels() {
  try {
    const res = await modelRatingAPI.getAllScores()
    if (res.code === 0 && Array.isArray(res.data)) {
      const uniqueModels = new Map()
      res.data.forEach(item => {
        const baseName = normalizeModelName(item.model_name)
        const normalizedName = baseName.charAt(0).toUpperCase() + baseName.substring(1)
        if (baseName && !uniqueModels.has(normalizedName)) {
          uniqueModels.set(normalizedName, {
            id: item.model_key,
            name: normalizedName,
            originalName: item.model_name,
            channel_name: item.channel_name,
            score: item.score
          })
        }
      })
      models.value = Array.from(uniqueModels.values()).sort((a, b) => b.score - a.score)
    }
  } catch (e) {
    console.error('加载模型失败', e)
    try {
      const fallback = await modelAPI.list({ page: 1, page_size: 100 })
      if (fallback.code === 0) {
        models.value = fallback.data?.items || []
      }
    } catch (e2) {
      console.error('备用加载也失败', e2)
    }
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await tokenAPI.list({ page: page.value, page_size: pageSize.value })
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

function showDialog() {
  isEdit.value = false
  editingId.value = null
  form.name = ''
  form.format = 'openai'
  form.type = 'chat'
  form.model_name = ''
  form.total_token_limit = 0
  form.expires_at = null
  rateLimitRules.value = []
  dialogVisible.value = true
}

function editToken(row) {
  isEdit.value = true
  editingId.value = row.id
  form.name = row.name
  form.format = row.format
  form.type = row.type
  form.model_name = row.model_name
  form.total_token_limit = row.total_token_limit || 0
  form.expires_at = row.expires_at || null
  rateLimitRules.value = parseRateLimits(row.rate_limits)
  dialogVisible.value = true
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  try {
    const submitData = {
      ...form,
      rate_limits: JSON.stringify(rateLimitRules.value)
    }
    let res
    if (isEdit.value) {
      res = await tokenAPI.update(editingId.value, submitData)
      if (res.code === 0) {
        ElMessage.success('保存成功')
        dialogVisible.value = false
        loadData()
      } else {
        ElMessage.error(res.message || '保存失败')
      }
    } else {
      res = await tokenAPI.create(submitData)
      if (res.code === 0) {
        createdKey.value = res.data.key
        dialogVisible.value = false
        resultDialogVisible.value = true
        loadData()
      } else {
        ElMessage.error(res.message || '创建失败')
      }
    }
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function toggleEnabled(row) {
  try {
    await tokenAPI.setEnabled(row.id, row.enabled ? 0 : 1)
    ElMessage.success(row.enabled ? '已禁用' : '已启用')
    loadData()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(row) {
  await ElMessageBox.confirm('确定要删除该API吗？此操作不可恢复', '警告', { type: 'warning' })
  try {
    await tokenAPI.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

function copyKey(key) {
  navigator.clipboard.writeText(key)
  ElMessage.success('已复制到剪贴板')
}

function parseRateLimits(rateLimitsStr) {
  try {
    return JSON.parse(rateLimitsStr || '[]')
  } catch {
    return []
  }
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
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

code {
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.rate-limit-rules {
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rate-limit-rule {
  display: flex;
  align-items: center;
  gap: 8px;
}

.form-help {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}
</style>
