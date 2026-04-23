<template>
  <router-view />
  <SetupDialog v-if="showSetup" />
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SetupDialog from './components/SetupDialog.vue'
import axios from 'axios'

const showSetup = ref(false)

onMounted(async () => {
  try {
    const response = await axios.get('/api/auth/setup-status')
    const data = response.data.data
    if (!data.password_setup_done) {
      showSetup.value = true
    }
  } catch (e) {
    console.error('Failed to check setup status:', e)
  }
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif;
}

#app {
  min-height: 100vh;
}
</style>
