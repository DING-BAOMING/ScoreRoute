<template>
  <div class="other-rating">
    <el-row :gutter="20">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>成本评分规则</span>
          </template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="周期计费模型">
              <el-tag type="success">100分</el-tag>
              <span class="rule-desc">（按固定周期计费，如*元/天、周、月、季度、年）</span>
            </el-descriptions-item>
            <el-descriptions-item label="免费模型">
              <el-tag type="success">90分</el-tag>
              <span class="rule-desc">（CostPerToken = 0）</span>
            </el-descriptions-item>
            <el-descriptions-item label="Token/次数计费模型">
              <div>最低价（最便宜）: <el-tag type="warning">70分</el-tag></div>
              <div>最高价（最贵）: <el-tag type="danger">1分</el-tag></div>
              <span class="rule-desc">（中间按比例计算）</span>
            </el-descriptions-item>
            <el-descriptions-item label="特殊情况">
              <span>只有一个收费模型或所有价格相同时: </span><el-tag>50分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="货币转换">
              <span>使用汇率将USD转换为CNY统一计算</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <span>时间评分规则</span>
          </template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="≤7天">
              <el-tag type="success">100分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="≤30天">
              <el-tag type="success">90-99分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="≤60天">
              <el-tag type="warning">80-89分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="≤120天">
              <el-tag type="warning">70-79分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="≤180天">
              <el-tag type="warning">60-69分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="≤365天">
              <el-tag type="danger">1-59分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="＞365天/已过期">
              <el-tag type="danger">0分</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="永不过期">
              <el-tag type="info">70分</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>成本/时间评分详情</span>
          <el-button type="primary" @click="loadCostTimeRatings">
            <el-icon><RefreshRight /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-table :data="costTimeRatings" v-loading="loading" stripe>
        <el-table-column prop="model_key" label="模型标识" width="300" />
        <el-table-column prop="cost_per_token" label="成本/千Token" width="120">
          <template #default="{ row }">
            <span v-if="row.cost_per_token > 0">{{ row.cost_per_token }} {{ row.currency }}</span>
            <el-tag v-else type="success" size="small">免费</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cost_rating" label="成本评分" width="100">
          <template #default="{ row }">
            <el-tag :type="getCostTagType(row.cost_rating)">{{ row.cost_rating }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="days_left" label="剩余天数" width="100">
          <template #default="{ row }">
            <span v-if="row.days_left > 0">{{ row.days_left }}天</span>
            <span v-else-if="row.expires_at">已过期</span>
            <el-tag v-else type="info" size="small">永不过期</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time_rating" label="时间评分" width="100">
          <template #default="{ row }">
            <el-tag :type="getTimeTagType(row.time_rating)">{{ row.time_rating }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="180">
          <template #default="{ row }">
            <span v-if="row.expires_at">{{ row.expires_at }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { modelRatingAPI } from '../api'

const loading = ref(false)
const costTimeRatings = ref([])

onMounted(() => {
  loadCostTimeRatings()
})

async function loadCostTimeRatings() {
  loading.value = true
  try {
    const res = await modelRatingAPI.getCostTimeRatings()
    if (res.code === 0 && res.data) {
      costTimeRatings.value = res.data
    }
  } catch (e) {
    ElMessage.error('加载评分详情失败')
  } finally {
    loading.value = false
  }
}

function getCostTagType(score) {
  if (score >= 80) return 'success'
  if (score >= 50) return 'warning'
  return 'danger'
}

function getTimeTagType(score) {
  if (score >= 80) return 'success'
  if (score >= 50) return 'warning'
  return 'danger'
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.rule-desc {
  color: #909399;
  font-size: 12px;
  margin-left: 5px;
}
</style>
