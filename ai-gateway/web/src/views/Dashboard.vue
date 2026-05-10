<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.today_tokens || 0) }}</div>
              <div class="stat-label">今日Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a">
              <el-icon><Calendar /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.week_tokens || 0) }}</div>
              <div class="stat-label">本周Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #e6a23c">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatNumber(stats.month_tokens || 0) }}</div>
              <div class="stat-label">本月Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #f56c6c">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ Math.round(stats.avg_latency || 0) }}ms</div>
              <div class="stat-label">平均延迟</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #909399">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ channelCount }}</div>
              <div class="stat-label">渠道数量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ tokenCount }}</div>
              <div class="stat-label">API数量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.today_calls || 0 }}</div>
              <div class="stat-label">今日调用</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #e6a23c">
              <el-icon><Document /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ stats.total_calls || 0 }}</div>
              <div class="stat-label">总调用次数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>渠道调用排行</span>
          </template>
          <el-table :data="topChannels" stripe>
            <el-table-column prop="channel_name" label="渠道名称" />
            <el-table-column prop="call_count" label="调用次数" width="100" />
            <el-table-column prop="avg_latency" label="平均延迟" width="100">
              <template #default="{ row }">
                {{ row.avg_latency ? Math.round(row.avg_latency) + 'ms' : '-' }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>API调用统计</span>
          </template>
          <el-table :data="tokenStats" stripe>
            <el-table-column prop="token_name" label="API名称" />
            <el-table-column prop="total_calls" label="总调用" width="80" />
            <el-table-column prop="today_calls" label="今日" width="70" />
            <el-table-column prop="week_calls" label="本周" width="70" />
            <el-table-column prop="month_calls" label="本月" width="70" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>安全设置</span>
            </div>
          </template>
          <div class="security-settings">
            <div class="setting-item">
              <div class="setting-info">
                <div class="setting-title">
                  <el-icon><Lock /></el-icon>
                  <span>无需密码访问</span>
                </div>
                <div class="setting-desc">
                  启用后可直接访问系统，无需登录（适合开发测试环境）
                </div>
              </div>
              <el-switch
                v-model="passwordLessMode"
                @change="handlePasswordLessModeChange"
              />
            </div>
            <div class="setting-item">
              <div class="setting-info">
                <div class="setting-title">
                  <el-icon><Key /></el-icon>
                  <span>修改密码</span>
                </div>
                <div class="setting-desc">
                  更改管理员登录密码
                </div>
              </div>
              <el-button type="primary" size="small" @click="showPasswordDialog = true">
                修改密码
              </el-button>
            </div>
            <el-alert
              v-if="passwordLessMode"
              type="warning"
              :closable="false"
              show-icon
              class="security-warning"
            >
              <template #title>
                安全提醒：当前已启用无需密码访问，任何人都可以访问系统管理界面。
                如需更高安全性，请关闭此功能。
              </template>
            </el-alert>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="showPasswordDialog" title="修改密码" width="420px">
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="85px" status-icon>
        <el-form-item label="新密码" prop="password">
          <el-input v-model="passwordForm.password" type="password" show-password placeholder="请输入新密码（至少6位）" />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPasswordDialog = false">取消</el-button>
        <el-button type="primary" @click="handleChangePassword">确定</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { logAPI, channelAPI, tokenAPI } from '../api'

const stats = ref({})
const topChannels = ref([])
const tokenStats = ref([])
const channelCount = ref(0)
const tokenCount = ref(0)
const passwordLessMode = ref(false)
const showPasswordDialog = ref(false)
const passwordFormRef = ref()
const passwordForm = ref({
  password: '',
  confirmPassword: ''
})

function formatNumber(num) {
  if (!num) return '0'
  if (num >= 100000000) {
    return (num / 100000000).toFixed(1).replace(/\.0$/, '') + '亿'
  }
  if (num >= 10000) {
    return (num / 10000).toFixed(1).replace(/\.0$/, '') + '万'
  }
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

onMounted(async () => {
  await loadDashboard()
  await loadSecuritySettings()
})

const passwordRules = {
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { 
      validator: (rule, value, callback) => {
        if (value !== passwordForm.value.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

async function loadSecuritySettings() {
  try {
    const response = await logAPI.getSystemConfig()
    if (response.code === 0) {
      passwordLessMode.value = response.data.password_less_mode || false
    }
  } catch (e) {
    console.error('Failed to load security settings:', e)
  }
}

async function handlePasswordLessModeChange(value) {
  try {
    await logAPI.setPasswordLessMode({ enabled: value })
    localStorage.setItem('password_less_mode', value ? 'true' : 'false')
    if (!value) {
      localStorage.removeItem('token')
      ElMessage.warning('已禁用无需密码访问，3秒后跳转到登录页...')
      setTimeout(() => { window.location.href = '/login' }, 3000)
    } else {
      ElMessage.success('已启用无需密码访问')
    }
  } catch (e) {
    ElMessage.error('设置失败：' + (e.message || '未知错误'))
    passwordLessMode.value = !value
  }
}

async function handleChangePassword() {
  const valid = await passwordFormRef.value.validate().catch(() => false)
  if (!valid) { ElMessage.warning("请检查输入格式"); return }
  
  try {
    await logAPI.changePassword({ password: passwordForm.value.password })
    ElMessage.success('密码修改成功')
    showPasswordDialog.value = false
    passwordForm.value.password = ''
    passwordForm.value.confirmPassword = ''
  } catch (e) {
    ElMessage.error('修改失败：' + (e.message || '未知错误'))
  }
}

async function loadDashboard() {
  try {
    const [dashboardRes, channelsRes, tokensRes] = await Promise.all([
      logAPI.dashboard(),
      channelAPI.list({ page: 1, page_size: 1 }),
      tokenAPI.list({ page: 1, page_size: 1 })
    ])
    
    if (dashboardRes.code === 0) {
      stats.value = dashboardRes.data?.items?.stats || {}
      topChannels.value = dashboardRes.data?.items?.top_channels || []
      tokenStats.value = dashboardRes.data?.items?.token_stats || []
    }
    
    if (channelsRes.code === 0) {
      channelCount.value = channelsRes.data?.total || 0
    }

    if (tokensRes.code === 0) {
      tokenCount.value = tokensRes.data?.total || 0
    }
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 20px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 24px;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #333;
}

.stat-label {
  color: #999;
  font-size: 14px;
  margin-top: 5px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.security-settings {
  padding: 10px 0;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 0;
  border-bottom: 1px solid #ebeef5;
}

.setting-item:last-of-type {
  border-bottom: none;
}

.setting-info {
  flex: 1;
}

.setting-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.setting-desc {
  font-size: 13px;
  color: #909399;
  margin-top: 5px;
  margin-left: 32px;
}

.security-warning {
  margin-top: 15px;
}

</style>
