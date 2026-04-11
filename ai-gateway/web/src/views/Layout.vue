<template>
  <el-container class="layout-container">
    <el-aside width="200px" class="aside">
      <div class="logo">AI Gateway</div>
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
        <el-menu-item index="/docs">
          <el-icon><Document /></el-icon>
          <span>开发文档</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">API网关管理系统</div>
        <div class="header-actions">
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
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const activeMenu = computed(() => route.path)

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
}

.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  background: #2b3a4a;
}

.menu {
  border-right: none;
  background: transparent;
}

:deep(.el-menu-item) {
  color: #bfcbd9;
}

:deep(.el-menu-item:hover),
:deep(.el-menu-item.is-active) {
  background: #263445;
  color: #409eff;
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
