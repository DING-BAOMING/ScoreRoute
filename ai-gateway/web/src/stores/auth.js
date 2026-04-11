import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authAPI } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref('')

  const isLoggedIn = !!token.value

  async function login(user, pass) {
    const res = await authAPI.login({ username: user, password: pass })
    if (res.code === 0) {
      token.value = res.data.token
      username.value = user
      localStorage.setItem('token', res.data.token)
      return true
    }
    throw new Error(res.message)
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
  }

  async function validate() {
    try {
      const res = await authAPI.validate()
      return res.code === 0
    } catch {
      return false
    }
  }

  return {
    token,
    username,
    isLoggedIn,
    login,
    logout,
    validate
  }
})
