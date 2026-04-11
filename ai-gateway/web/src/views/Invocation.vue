<template>
  <div class="invocation">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>模型调用视图</span>
          <el-tag type="success">共 {{ totalModels }} 个可用模型</el-tag>
        </div>
      </template>

      <el-collapse v-model="activeNames">
        <el-collapse-item v-for="group in groupedModels" :key="group.key" :name="group.key">
          <template #title>
            <div class="group-title">
              <el-tag type="primary">{{ group.format }}</el-tag>
              <el-tag type="info" style="margin-left: 8px">{{ group.type }}</el-tag>
              <span style="margin-left: 12px; color: #909399">{{ group.models.length }} 个模型</span>
            </div>
          </template>
          
          <el-table :data="group.models" stripe style="width: 100%">
            <el-table-column prop="channel_name" label="渠道" width="150" />
            <el-table-column prop="name" label="模型名称" />
            <el-table-column prop="type" label="类型" width="100">
              <template #default="{ row }">
                <el-tag type="info">{{ row.type }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="call_count" label="调用次数" width="100" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
                  {{ row.enabled ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button 
                  size="small" 
                  :type="row.enabled ? 'danger' : 'success'" 
                  @click="toggleModel(row)"
                  :loading="row.loading"
                >
                  {{ row.enabled ? '禁用' : '启用' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-collapse-item>
      </el-collapse>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { modelAPI } from '../api'

const loading = ref(false)
const models = ref([])
const total = ref(0)
const activeNames = ref([])

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const res = await modelAPI.list({ page: 1, page_size: 500 })
    if (res.code === 0) {
      const items = res.data?.items || []
      models.value = items.map(m => ({ ...m, loading: false }))
      total.value = res.data?.total || 0
      
      activeNames.value = groupedModels.value.map(g => g.key)
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const groupedModels = computed(() => {
  const groups = {}
  
  for (const model of models.value) {
    const key = `${model.format || 'unknown'}_${model.type || 'chat'}`
    if (!groups[key]) {
      groups[key] = {
        key: key,
        format: (model.format || 'unknown').toUpperCase(),
        type: model.type || 'chat',
        models: []
      }
    }
    groups[key].models.push(model)
  }
  
  return Object.values(groups).sort((a, b) => {
    if (a.format !== b.format) return a.format.localeCompare(b.format)
    return a.type.localeCompare(b.type)
  })
})

const totalModels = computed(() => models.value.filter(m => m.enabled).length)

async function toggleModel(row) {
  row.loading = true
  try {
    await modelAPI.setEnabled(row.id, row.enabled ? 0 : 1)
    row.enabled = row.enabled ? 0 : 1
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    row.loading = false
  }
}
</script>

<style scoped>
.invocation {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.group-title {
  display: flex;
  align-items: center;
  font-size: 16px;
}
</style>
