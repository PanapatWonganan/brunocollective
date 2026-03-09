import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')

  const isLoggedIn = computed(() => !!token.value)

  async function login(user: string, password: string) {
    const { data } = await api.post('/login', { username: user, password })
    token.value = data.token
    username.value = data.user.username
    localStorage.setItem('token', data.token)
    localStorage.setItem('username', data.user.username)
    router.push('/')
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    router.push('/login')
  }

  return { token, username, isLoggedIn, login, logout }
})
