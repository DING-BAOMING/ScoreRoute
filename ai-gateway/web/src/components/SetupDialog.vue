<template>
  <el-dialog
    v-model="dialogVisible"
    title="初始设置"
    width="500px"
    :close-on-click-modal="false"
    :show-close="false"
    draggable
  >
    <div class="setup-content">
      <el-steps :active="step" finish-status="success" align-center>
        <el-step title="选择模式" />
        <el-step title="完成" />
      </el-steps>
      
      <div v-if="step === 0" class="step-content">
        <p class="intro-text">欢迎使用 ScoreRoute API 网关管理系统</p>
        <p class="intro-text">请选择访问模式：</p>
        
        <div class="mode-options">
          <el-card class="mode-card" shadow="hover" @click="selectMode('password')">
            <template #header>
              <div class="mode-header">
                <el-icon size="24"><Lock /></el-icon>
                <span>设置密码</span>
              </div>
            </template>
            <div class="mode-desc">
              需要输入密码才能访问系统<br>
              <small>更安全，适合生产环境</small>
            </div>
          </el-card>
          
          <el-card class="mode-card" shadow="hover" @click="selectMode('passwordless')">
            <template #header>
              <div class="mode-header">
                <el-icon size="24"><Key /></el-icon>
                <span>无需密码</span>
              </div>
            </template>
            <div class="mode-desc">
              直接访问，无需登录<br>
              <small>更便捷，适合开发测试</small>
            </div>
          </el-card>
        </div>
        
        <el-alert
          v-if="mode === 'passwordless'"
          type="warning"
          :closable="false"
          show-icon
          class="warning-alert"
        >
          <template #title>
            安全提示：无需密码模式意味着任何人都可以访问系统管理界面。
            请确保在可信赖的网络环境下使用。
          </template>
        </el-alert>
        
        <div v-if="mode === 'password'" class="password-setup">
          <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef">
            <el-form-item prop="password">
              <el-input
                v-model="passwordForm.password"
                type="password"
                placeholder="请输入密码（至少6位）"
                show-password
                size="large"
              />
            </el-form-item>
            <el-form-item prop="confirmPassword">
              <el-input
                v-model="passwordForm.confirmPassword"
                type="password"
                placeholder="请确认密码"
                show-password
                size="large"
              />
            </el-form-item>
          </el-form>
        </div>
      </div>
      
      <div v-if="step === 1" class="step-content success-content">
        <el-result
          icon="success"
          :title="successTitle"
          :sub-title="successSubTitle"
        />
      </div>
    </div>
    
    <template #footer>
      <div v-if="step === 0">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleNext" :disabled="!canProceed">
          下一步
        </el-button>
      </div>
      <div v-else>
        <el-button type="primary" @click="handleFinish">开始使用</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Lock, Key } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()

const dialogVisible = ref(true)
const step = ref(0)
const mode = ref('')
const passwordFormRef = ref()
const passwordForm = ref({
  password: '',
  confirmPassword: ''
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

const successTitle = computed(() => {
  if (mode.value === 'passwordless') {
    return '设置完成'
  }
  return '密码已设置'
})

const successSubTitle = computed(() => {
  if (mode.value === 'passwordless') {
    return '已启用无需密码访问模式，您可以直接使用系统'
  }
  return '已设置密码，下次登录时需要输入密码'
})

const canProceed = computed(() => {
  if (mode.value === 'passwordless') {
    return true
  }
  if (mode.value === 'password') {
    return passwordForm.value.password.length >= 6 && 
           passwordForm.value.password === passwordForm.value.confirmPassword
  }
  return false
})

function selectMode(selectedMode) {
  mode.value = selectedMode
}

async function handleNext() {
  if (mode.value === 'passwordless') {
    try {
      await axios.put('/api/system-config/password-less', { enabled: true })
      step.value = 1
    } catch (e) {
      ElMessage.error('设置失败：' + (e.message || '未知错误'))
    }
  } else if (mode.value === 'password') {
    const valid = await passwordFormRef.value.validate().catch(() => false)
    if (!valid) return
    
    try {
      await axios.post('/api/system-config/setup-password', {
        password: passwordForm.value.password
      })
      step.value = 1
    } catch (e) {
      ElMessage.error('设置失败：' + (e.message || '未知错误'))
    }
  }
}

function handleCancel() {
  dialogVisible.value = false
  router.push('/')
}

async function handleFinish() {
  dialogVisible.value = false
  if (mode.value === 'passwordless') {
    try {
      const response = await axios.post('/api/auth/passwordless-login')
      localStorage.setItem('token', response.data.data.token)
      ElMessage.success('登录成功')
      router.push('/')
    } catch (e) {
      ElMessage.error('登录失败：' + (e.message || '未知错误'))
    }
  } else {
    dialogVisible.value = false
    router.push('/login')
  }
}
</script>

<style scoped>
.setup-content {
  padding: 20px 0;
}

.step-content {
  margin-top: 30px;
}

.intro-text {
  text-align: center;
  color: #606266;
  margin: 10px 0;
}

.mode-options {
  display: flex;
  gap: 20px;
  margin: 30px 0;
}

.mode-card {
  flex: 1;
  cursor: pointer;
  transition: all 0.3s;
}

.mode-card:hover {
  transform: translateY(-5px);
}

.mode-header {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: bold;
}

.mode-desc {
  color: #909399;
  font-size: 14px;
  line-height: 1.6;
}

.mode-desc small {
  color: #c0c4cc;
}

.warning-alert {
  margin-top: 20px;
}

.password-setup {
  margin-top: 20px;
}

.success-content {
  padding: 20px 0;
}
</style>
