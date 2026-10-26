import { ref } from 'vue'
import { defineStore } from 'pinia'

export interface Toast {
  id: number
  text: string
  kind: 'info' | 'error'
}

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  let seq = 0

  function toast(text: string, kind: 'info' | 'error' = 'info') {
    const id = ++seq
    toasts.value.push({ id, text, kind })
    window.setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, 2600)
  }

  return { toasts, toast }
})
