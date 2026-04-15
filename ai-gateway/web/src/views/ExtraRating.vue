<template>
  <div class="extra-rating">
    <el-card style="margin-bottom: 20px">
      <template #header>
        <span>规则说明</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="惩罚机制">
          当某个<el-tag type="danger" size="small">特定模型</el-tag>被调用时，该模型会获得一个新的惩罚分数（实现模型轮转调用）
        </el-descriptions-item>
        <el-descriptions-item label="惩罚衰减">
          每当该模型被再次调用时，其所有现有惩罚分数都会减少1分，低于0分时自动清除
        </el-descriptions-item>
        <el-descriptions-item label="惩罚叠加">
          多次调用同一模型可以叠加惩罚分数，惩罚持续期间可以累加
        </el-descriptions-item>
        <el-descriptions-item label="奖励机制">
          当<el-tag type="success" size="small">新模型</el-tag>被添加到系统时，该模型会获得奖励分数（吸引使用新加入的模型）
        </el-descriptions-item>
        <el-descriptions-item label="奖励衰减">
          奖励是基于时间的线性衰减，持续24小时（可配置），每小时衰减约1/24
        </el-descriptions-item>
        <el-descriptions-item label="模型标识">
          奖励和惩罚的模型以 <el-tag size="small">API渠道名_格式_类型_模型名</el-tag> 为一个单位
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>额外评分配置</span>
          <el-button type="primary" @click="loadConfig">
            <el-icon><RefreshRight /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-form :model="configForm" label-width="140px" style="max-width: 600px">
        <el-form-item label="惩罚轮数">
          <el-input-number v-model="configForm.punishment_rounds" :min="1" :max="100" />
          <span class="form-help">惩罚持续次数（每次请求后递减）</span>
        </el-form-item>
        <el-form-item label="每次惩罚分数">
          <el-input-number v-model="configForm.punishment_score" :min="1" :max="100" />
          <span class="form-help">每次调用模型时扣除的分数</span>
        </el-form-item>
        <el-form-item label="奖励小时数">
          <el-input-number v-model="configForm.reward_hours" :min="1" :max="168" />
          <span class="form-help">奖励持续时间（小时）</span>
        </el-form-item>
        <el-form-item label="每次奖励分数">
          <el-input-number v-model="configForm.reward_score" :min="1" :max="100" />
          <span class="form-help">调用后获得的奖励分数</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>惩罚/奖励记录</span>
          <div>
            <el-button type="danger" @click="handleClearAll" :loading="clearing">
              清空所有记录
            </el-button>
            <el-button type="primary" @click="loadRecords">
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="惩罚记录" name="penalty">
          <el-table :data="penaltyRecords" v-loading="loading" stripe>
            <el-table-column prop="model_key" label="模型标识" width="250" />
            <el-table-column prop="penalty_score" label="原始惩罚" width="100" />
            <el-table-column prop="current_score" label="当前分数" width="100">
              <template #default="{ row }">
                <el-tag type="danger">{{ row.current_score }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="decay_per_request" label="每次递减" width="100" />
            <el-table-column prop="request_count" label="请求次数" width="100" />
            <el-table-column label="剩余有效期" width="140">
              <template #default="{ row }">
                <span v-if="row.expires_at">{{ formatExpiry(row.expires_at) }}</span>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="奖励记录" name="reward">
          <el-table :data="rewardRecords" v-loading="loading" stripe>
            <el-table-column prop="model_key" label="模型标识" width="250" />
            <el-table-column prop="reward_score" label="原始奖励" width="100" />
            <el-table-column prop="current_score" label="当前分数" width="100">
              <template #default="{ row }">
                <el-tag type="success">{{ row.current_score }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="decay_per_request" label="每次递减" width="100" />
            <el-table-column prop="request_count" label="请求次数" width="100" />
            <el-table-column label="剩余有效期" width="140">
              <template #default="{ row }">
                <span v-if="row.expires_at">{{ formatExpiry(row.expires_at) }}</span>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>

      <el-row :gutter="20" style="margin-top: 20px">
        <el-col :span="8">
          <el-statistic title="惩罚记录总数" :value="penaltyRecords.length" />
        </el-col>
        <el-col :span="8">
          <el-statistic title="奖励记录总数" :value="rewardRecords.length" />
        </el-col>
        <el-col :span="8">
          <el-statistic title="当前总惩罚" :value="totalPenalty" />
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { extraRatingAPI } from '../api'

const loading = ref(false)
const saving = ref(false)
const clearing = ref(false)
const activeTab = ref('penalty')

const configForm = ref({
  punishment_rounds: 5,
  punishment_score: 5,
  reward_hours: 24,
  reward_score: 5
})

const penaltyRecords = ref([])
const rewardRecords = ref([])

const totalPenalty = computed(() => {
  return penaltyRecords.value.reduce((sum, r) => sum + r.current_score, 0)
})

onMounted(() => {
  loadConfig()
  loadRecords()
})

async function loadConfig() {
  try {
    const res = await extraRatingAPI.getConfig()
    if (res.code === 0 && res.data) {
      configForm.value = {
        punishment_rounds: res.data.punishment_rounds || 5,
        punishment_score: res.data.punishment_score || 5,
        reward_hours: res.data.reward_hours || 24,
        reward_score: res.data.reward_score || 5
      }
    }
  } catch (e) {
    ElMessage.error('加载配置失败')
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const res = await extraRatingAPI.setConfig(configForm.value)
    if (res.code === 0) {
      ElMessage.success('保存成功')
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

async function loadRecords() {
  loading.value = true
  try {
    const res = await extraRatingAPI.getRecords()
    if (res.code === 0 && res.data) {
      penaltyRecords.value = res.data.penalty_records || []
      rewardRecords.value = res.data.reward_records || []
    }
  } catch (e) {
    ElMessage.error('加载记录失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(id) {
  try {
    await ElMessageBox.confirm('确定要删除这条记录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await extraRatingAPI.deleteRecord(id)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      loadRecords()
    } else {
      ElMessage.error(res.message || '删除失败')
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function handleClearAll() {
  try {
    await ElMessageBox.confirm('确定要清空所有记录吗？此操作不可恢复！', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    clearing.value = true
    const res = await extraRatingAPI.clearRecords()
    if (res.code === 0) {
      ElMessage.success('清空成功')
      loadRecords()
    } else {
      ElMessage.error(res.message || '清空失败')
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('清空失败')
    }
  } finally {
    clearing.value = false
  }
}

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

function formatExpiry(timeStr) {
  if (!timeStr) return '-'
  const expiry = new Date(timeStr)
  const now = new Date()
  const diff = expiry - now
  if (diff <= 0) return '已过期'
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  if (hours > 0) {
    return `${hours}小时${minutes}分钟`
  }
  return `${minutes}分钟`
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-help {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}
</style>
