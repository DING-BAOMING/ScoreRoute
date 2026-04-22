<template>
  <el-container class="layout-container">
    <el-aside width="200px" class="aside">
      <div class="logo"><img src="/logo.png" alt="ScoreRoute" style="height: 40px; margin-top: 10px;"></div>
      <el-menu :default-active="activeMenu" router class="menu">
        <el-menu-item index="/">
          <el-icon><House /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/channels">
          <el-icon><Connection /></el-icon>
          <span>接入管理</span>
        </el-menu-item>
        <el-menu-item index="/models">
          <el-icon><Box /></el-icon>
          <span>模型管理</span>
        </el-menu-item>
        <el-menu-item index="/tokens">
          <el-icon><Key /></el-icon>
          <span>接出管理</span>
        </el-menu-item>
        <el-menu-item index="/invocation">
          <el-icon><Connection /></el-icon>
          <span>模型调用</span>
        </el-menu-item>
        <el-menu-item index="/logs">
          <el-icon><Document /></el-icon>
          <span>调用日志</span>
        </el-menu-item>
        <el-menu-item index="/model-rating">
          <el-icon><DataAnalysis /></el-icon>
          <span>模型评分</span>
        </el-menu-item>
        <el-menu-item index="/user-rating">
          <el-icon><User /></el-icon>
          <span>用户评分</span>
        </el-menu-item>
        <el-menu-item index="/sample-analysis">
          <el-icon><Document /></el-icon>
          <span>样本分析</span>
        </el-menu-item>
        <el-menu-item index="/extra-rating">
          <el-icon><DataAnalysis /></el-icon>
          <span>额外评分</span>
        </el-menu-item>
        <el-menu-item index="/other-rating">
          <el-icon><DataAnalysis /></el-icon>
          <span>其他评分</span>
        </el-menu-item>
        <el-menu-item index="/docs">
          <el-icon><Document /></el-icon>
          <span>开发文档</span>
        </el-menu-item>
      </el-menu>
      <div class="version-info">
        <div class="version-text">v{{ currentVersion }}</div>
        <div v-if="hasNewVersion" class="version-badge">
          <el-badge value="新" type="danger" @click="openNewVersionUrl">
            <span class="new-version-text">发现新版本</span>
          </el-badge>
        </div>
      </div>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">API网关管理系统</div>
        <div class="header-actions">
          <el-button type="primary" link @click="openUrl('https://www.scoreroute.com/index.html')">
            <el-icon><Link /></el-icon>
            官网
          </el-button>
          <el-button type="primary" link @click="openUrl('https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=82bq51e8-fbd2-4c36-97d9-4e9f575c3d1b')">
            <el-icon><ChatDotRound /></el-icon>
            飞书群
          </el-button>
          <el-button type="primary" link @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            退出登录
          </el-button>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox, ElNotification } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const currentVersion = ref('2.19.0')
const hasNewVersion = ref(false)
const latestVersion = ref('')
let checkInterval = null

const activeMenu = computed(() => route.path)

const GITHUB_RELEASES_URL = 'https://api.github.com/repos/DING-BAOMING/ScoreRoute/releases/latest'
const ONE_DAY_MS = 86400000

function openUrl(url) {
  window.open(url, "_blank")
}

function openNewVersionUrl() {
  window.open('https://github.com/DING-BAOMING/ScoreRoute/releases', '_blank')
}

async function checkNewVersion() {
  try {
    const response = await fetch(GITHUB_RELEASES_URL)
    if (!response.ok) return
    
    const data = await response.json()
    const tagName = data.tag_name || ''
    const version = tagName.startsWith('v') ? tagName.substring(1) : tagName
    
    latestVersion.value = version
    
    const currentParts = currentVersion.value.split('.').map(Number)
    const latestParts = version.split('.').map(Number)
    
    let isNewer = false
    for (let i = 0; i < Math.max(currentParts.length, latestParts.length); i++) {
      const curr = currentParts[i] || 0
      const latest = latestParts[i] || 0
      if (latest > curr) {
        isNewer = true
        break
      }
      if (latest < curr) break
    }
    
    if (isNewer && !localStorage.getItem('versionDismissed_' + version)) {
      hasNewVersion.value = true
      ElNotification({
        title: '发现新版本',
        message: `检测到新版本 v${version}，点击查看更新`,
        type: 'info',
        duration: 0,
        onClick: () => openNewVersionUrl()
      })
    }
    
    localStorage.setItem('lastVersionCheck', Date.now().toString())
  } catch (e) {
    console.error('Version check failed:', e)
  }
}

function startVersionPolling() {
  const lastCheck = localStorage.getItem('lastVersionCheck')
  const now = Date.now()
  
  if (!lastCheck || (now - parseInt(lastCheck)) >= ONE_DAY_MS) {
    checkNewVersion()
  } else {
    const nextCheckDelay = ONE_DAY_MS - (now - parseInt(lastCheck))
    setTimeout(() => {
      checkNewVersion()
      checkInterval = setInterval(checkNewVersion, ONE_DAY_MS)
    }, nextCheckDelay)
  }
}

onMounted(() => {
  startVersionPolling()
})

onUnmounted(() => {
  if (checkInterval) {
    clearInterval(checkInterval)
  }
})

function handleLogout() {
  ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    authStore.logout()
    router.push('/login')
  }).catch(() => {})
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.aside {
  background: #304156;
  display: flex;
  flex-direction: column;
}

.logo {
  height: 60px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #2b3a4a;
}

.menu {
  border-right: none;
  background: transparent;
  flex: 1;
}

:deep(.el-menu-item) {
  color: #bfcbd9;
}

:deep(.el-menu-item:hover),
:deep(.el-menu-item.is-active) {
  background: #263445;
  color: #409eff;
}

.version-info {
  padding: 10px;
  text-align: center;
  background: #2b3a4a;
  border-top: 1px solid #3d5066;
}

.version-text {
  color: #8a8a8a;
  font-size: 12px;
}

.version-badge {
  margin-top: 5px;
}

.new-version-text {
  color: #f56c6c;
  font-size: 12px;
  cursor: pointer;
}

.new-version-text:hover {
  text-decoration: underline;
}

.header {
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-title {
  font-size: 18px;
  font-weight: 500;
}

.main {
  background: #f0f2f5;
  padding: 20px;
}
</style>
