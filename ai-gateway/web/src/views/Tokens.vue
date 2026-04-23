<template>
  <div class="tokens-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>API密钥管理</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>
            创建Token
          </el-button>
        </div>
      </template>

      <el-table :data="tokenList" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="key" label="API Key" min-width="280">
          <template #default="{ row }">
            <div class="key-cell">
              <code>{{ row.key }}</code>
              <el-button size="small" text @click="copyKey(row.key)" title="复制">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button size="small" text type="primary" @click="showFullKey(row.id)" title="查看完整Key">
                <el-icon><View /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="model_name" label="模型" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.model_name === '__AUTO__'" type="success">自动</el-tag>
            <el-tag v-else type="info">{{ row.model_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="total_calls" label="调用次数" width="100" />
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="editToken(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click="deleteToken(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        @current-change="loadTokens"
        @size-change="loadTokens"
        style="margin-top: 20px"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑Token' : '创建Token'" width="600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="例如: 我的API" />
        </el-form-item>
        <el-form-item label="格式">
          <el-select v-model="form.format" placeholder="选择格式">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Zhipu" value="zhipu" />
            <el-option label="Claude" value="claude" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" placeholder="选择类型">
            <el-option label="Chat" value="chat" />
            <el-option label="Embedding" value="embedding" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型">
          <el-select v-model="form.model_name" placeholder="选择模型或输入" filterable allow-create>
            <el-option label="__AUTO__ (自动选择)" value="__AUTO__" />
            <el-option v-for="m in models" :key="m.id" :label="m.name" :value="m.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="限流规则">
          <div v-for="(rule, idx) in rateLimitRules" :key="idx" class="rate-limit-rule">
            <el-select v-model="rule.type" style="width: 100px">
              <el-option label="请求次数" value="calls" />
              <el-option label="Token数" value="tokens" />
            </el-select>
            <el-input-number v-model="rule.max_count" :min="1" style="width: 120px" />
            <el-select v-model="rule.window" style="width: 100px">
              <el-option label="分钟" value="minute" />
              <el-option label="小时" value="hour" />
              <el-option label="天" value="day" />
            </el-select>
            <el-button type="danger" link @click="removeRateLimit(idx)">删除</el-button>
          </div>
          <el-button type="primary" link @click="addRateLimit">+ 添加限流规则</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">{{ isEdit ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resultDialogVisible" title="Token创建成功" width="600px">
      <el-alert type="success" :closable="false" style="margin-bottom: 20px">
        <strong>请立即复制并妥善保存您的完整API Key！</strong><br/>
        关闭此对话框后，出于安全考虑，系统将不再显示完整Key
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="API地址">
          <el-input :value="apiUrl" readonly />
        </el-form-item>
        <el-form-item label="完整API Key">
          <el-input :value="createdKey" readonly class="full-key-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="copyKey(createdKey); resultDialogVisible = false">复制完整Key并关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="fullKeyDialogVisible" title="完整API Key" width="600px">
      <el-alert type="warning" :closable="false" style="margin-bottom: 20px">
        <strong>完整Key仅显示一次，请妥善保存！</strong>
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="Token名称">
          <el-input :value="fullKeyData.name" readonly />
        </el-form-item>
        <el-form-item label="完整API Key">
          <el-input :value="fullKeyData.key" readonly class="full-key-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="copyKey(fullKeyData.key); fullKeyDialogVisible = false">复制并关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { tokenAPI, modelAPI, modelRatingAPI } from '../api'
import { View, CopyDocument, Plus } from '@element-plus/icons-vue'

const loading = ref(false)
const tokenList = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const models = ref([])

const dialogVisible = ref(false)
const resultDialogVisible = ref(false)
const fullKeyDialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref(null)
const createdKey = ref('')
const apiUrl = ref('')
const fullKeyData = ref({ name: '', key: '' })

const form = reactive({
  name: '',
  format: 'openai',
  type: 'chat',
  model_name: '__AUTO__'
})

const rateLimitRules = ref([])

const rateLimitOptions = {
  calls: [
    { label: '分钟', value: 'minute' },
    { label: '小时', value: 'hour' },
    { label: '天', value: 'day' }
  ],
  tokens: [
    { label: '分钟', value: 'minute' },
    { label: '小时', value: 'hour' },
    { label: '天', value: 'day' }
  ]
}

function addRateLimit() {
  rateLimitRules.value.push({ type: 'calls', max_count: 100, window: 'minute' })
}

function removeRateLimit(idx) {
  rateLimitRules.value.splice(idx, 1)
}

async function loadTokens() {
  loading.value = true
  try {
    const res = await tokenAPI.list({ page: page.value, page_size: pageSize.value })
    if (res.code === 0) {
      tokenList.value = res.data.items
      total.value = res.data.total
    }
  } catch (e) {
    ElMessage.error('加载Token失败')
  } finally {
    loading.value = false
  }
}

async function loadModels() {
  try {
    const res = await modelAPI.list({ page: 1, page_size: 100 })
    if (res.code === 0) {
      models.value = res.data.items
    }
  } catch (e) {
    console.error('Failed to load models:', e)
  }
}

function openCreateDialog() {
  isEdit.value = false
  editingId.value = null
  form.name = ''
  form.format = 'openai'
  form.type = 'chat'
  form.model_name = '__AUTO__'
  rateLimitRules.value = []
  dialogVisible.value = true
}

function editToken(token) {
  isEdit.value = true
  editingId.value = token.id
  form.name = token.name
  form.format = token.format
  form.type = token.type
  form.model_name = token.model_name
  try {
    const limits = JSON.parse(token.rate_limits || '[]')
    rateLimitRules.value = limits.length > 0 ? limits : []
  } catch {
    rateLimitRules.value = []
  }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.name) {
    ElMessage.warning('请输入名称')
    return
  }
  try {
    const data = {
      name: form.name,
      format: form.format,
      type: form.type,
      model_name: form.model_name,
      enabled: 1,
      rate_limits: JSON.stringify(rateLimitRules.value)
    }
    let res
    if (isEdit.value) {
      res = await tokenAPI.update(editingId.value, data)
    } else {
      res = await tokenAPI.create(data)
    }
    if (res.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      if (!isEdit.value) {
        createdKey.value = res.data.key
        apiUrl.value = `${window.location.origin}/v1`
        resultDialogVisible.value = true
      }
      loadTokens()
    } else {
      ElMessage.error(res.message || '操作失败')
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function showFullKey(id) {
  try {
    const res = await tokenAPI.get(id)
    if (res.code === 0 && res.data) {
      fullKeyData.value = {
        name: res.data.name,
        key: res.data.key
      }
      fullKeyDialogVisible.value = true
    } else {
      ElMessage.error('获取Token详情失败')
    }
  } catch (e) {
    ElMessage.error('获取Token详情失败')
  }
}

function copyKey(key) {
  navigator.clipboard.writeText(key).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

async function deleteToken(id) {
  try {
    await ElMessageBox.confirm('确定要删除这个Token吗？删除后无法恢复', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await tokenAPI.delete(id)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      loadTokens()
    } else {
      ElMessage.error(res.message || '删除失败')
    }
  } catch {
  }
}

onMounted(() => {
  loadTokens()
  loadModels()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.key-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.key-cell code {
  font-family: Monaco, Consolas, monospace;
  color: #409eff;
}

.rate-limit-rule {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
  align-items: center;
}

.full-key-input :deep(.el-input__inner) {
  font-family: Monaco, Consolas, monospace;
  color: #409eff;
  font-weight: bold;
}
</style>
