import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { User } from '@/types'
import { api } from '@/api/http'

const TOKEN_KEY = 'mw_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) ?? '')
  const user = ref<User | null>(null)
  const ready = ref(false)

  function setSession(t: string, u: User) {
    token.value = t
    user.value = u
    localStorage.setItem(TOKEN_KEY, t)
  }

  function clear() {
    token.value = ''
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  async function init() {
    if (ready.value) return
    if (token.value) {
      try {
        const res = await api.get<{ user: User }>('/me')
        user.value = res.user
      } catch {
        clear()
      }
    }
    ready.value = true
  }

  async function login(username: string, password: string) {
    const res = await api.post<{ token: string; user: User }>('/auth/login', { username, password })
    setSession(res.token, res.user)
    ready.value = true
  }

  async function register(username: string, password: string, nickname: string) {
    const res = await api.post<{ token: string; user: User }>('/auth/register', {
      username,
      password,
      nickname,
    })
    setSession(res.token, res.user)
    ready.value = true
  }

  function updateUser(u: User) {
    user.value = u
  }

  function logout() {
    clear()
    ready.value = true
  }

  return { token, user, ready, init, login, register, updateUser, logout }
})
