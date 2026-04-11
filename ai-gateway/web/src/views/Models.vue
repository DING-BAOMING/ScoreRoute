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
        <el-table-column prop="channel_name" label="渠道" width="150">
          <template #default="{ row }">
            <el-tag type="info">{{ row.channel_name || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="模型名称" />
        <el-table-column prop="type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag type="info">{{ row.type || 'chat' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="call_count" label="调用次数" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280">
          <template #default="{ row }">
            <el-space :size="4">
              <el-button size="small" type="success" @click="testModel(row)">测试</el-button>
              <el-button size="small" @click="showDialog('edit', row)">编辑</el-button>
              <el-button size="small" type="danger" @click="toggleEnabled(row)">
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { modelAPI, channelAPI } from '../api'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const channels = ref([])
const filterChannel = ref('')

const dialogVisible = ref(false)
const batchDialogVisible = ref(false)
const dialogType = ref('create')
const formRef = ref()
const testing = ref(false)
const submitting = ref(false)

const form = reactive({
  id: null,
  channel_id: null,
  name: '',
  type: 'chat'
})

const batchForm = reactive({
  channel_id: null,
  model_names: [],
  type: 'chat'
})

const availableModels = ref([])
const fetchingModels = ref(false)

const rules = {
  channel_id: [{ required: true, message: '请选择渠道', trigger: 'change' }],
  name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

const dialogTitle = computed(() => dialogType.value === 'create' ? '添加模型' : '编辑模型')

onMounted(() => {
  loadChannels()
  loadData()
})

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
    Object.assign(form, { id: row.id, channel_id: row.channel_id, name: row.name, type: row.type || 'chat' })
  } else {
    Object.assign(form, { id: null, channel_id: null, name: '', type: 'chat' })
  }
  dialogVisible.value = true
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
</style>
