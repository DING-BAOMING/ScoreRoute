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
        <el-table-column prop="format" label="格式" width="120">
          <template #default="{ row }">
            <el-tag>{{ row.format }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="base_url" label="API地址" min-width="200" show-overflow-tooltip />
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
              <el-button size="small" type="success" @click="testChannel(row)" :loading="row.testing">测试</el-button>
              <el-button size="small" @click="showDialog('edit', row)">编辑</el-button>
              <el-button size="small" type="danger" @click="toggleEnabled(row)">
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
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
  api_key: ''
})

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

async function testChannel(row) {
  row.testing = true
  try {
    const res = await channelAPI.testCredentials({
      base_url: row.base_url,
      api_key: row.api_key
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
    Object.assign(form, row)
  } else {
    Object.assign(form, { name: '', format: 'openai', base_url: '', api_key: '' })
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
</style>
