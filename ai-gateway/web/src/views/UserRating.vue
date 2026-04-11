<template>
  <div class="user-rating">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户评分</span>
          <el-button type="primary" @click="loadData">
            <el-icon><RefreshRight /></el-icon>
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
            <li>用户评分范围: 1-100，默认为50</li>
            <li>评分会影响模型评分页面的最终评分计算</li>
            <li>相同模型名称（不区分大小写）的评分会同步到所有匹配渠道</li>
            <li>例如: minimaxai/minimax-m2.5 和 MiniMax-M2.5 会自动去重为 MiniMax-2.5</li>
          </ul>
        </template>
      </el-alert>
      
      <el-table :data="displayList" v-loading="loading" stripe>
        <el-table-column prop="model_name" label="模型名称" min-width="200">
          <template #default="{ row }">
            <span class="model-name">{{ row.model_name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="original_name" label="原始名称" min-width="200">
          <template #default="{ row }">
            <span class="original-name">{{ row.original_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="用户评分" width="220">
          <template #default="{ row }">
            <el-slider 
              v-model="row.user_rating" 
              :min="1" 
              :max="100" 
              :step="1"
              show-input
              :disabled="row.saving"
              @change="handleRatingChange(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button 
              type="primary" 
              size="small" 
              :loading="row.saving"
              :disabled="!row.dirty"
              @click="saveRating(row)">
              保存
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { userRatingAPI } from '../api'

const loading = ref(false)
const ratingList = ref([])

onMounted(() => {
  loadData()
})

async function loadData() {
  loading.value = true
  try {
    const res = await userRatingAPI.listDeduplicated()
    if (res.code === 0) {
      ratingList.value = (res.data || []).map(item => ({
        ...item,
        saving: false,
        dirty: false,
        original_rating: item.user_rating
      }))
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function handleRatingChange(row) {
  row.dirty = row.user_rating !== row.original_rating
}

async function saveRating(row) {
  row.saving = true
  try {
    const res = await userRatingAPI.upsert({
      model_name: row.model_name,
      user_rating: row.user_rating
    })
    if (res.code === 0) {
      row.original_rating = row.user_rating
      row.dirty = false
      ElMessage.success('保存成功')
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    row.saving = false
  }
}

const displayList = computed(() => {
  return ratingList.value
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.model-name {
  font-family: monospace;
  font-size: 13px;
  font-weight: bold;
}

.original-name {
  font-family: monospace;
  font-size: 12px;
  color: #909399;
}
</style>
