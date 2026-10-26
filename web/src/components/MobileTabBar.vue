<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import type { Tab } from '@/types'
import AppIcon from './AppIcon.vue'

defineProps<{ tab: Tab }>()
const emit = defineEmits<{ (e: 'change', tab: Tab): void }>()
const chat = useChatStore()

const items: { key: Tab; icon: string; label: string }[] = [
  { key: 'chats', icon: 'chat', label: '微信' },
  { key: 'contacts', icon: 'contacts', label: '通讯录' },
  { key: 'discover', icon: 'discover', label: '发现' },
  { key: 'me', icon: 'me', label: '我' },
]
</script>

<template>
  <nav class="mobile-tabs">
    <button
      v-for="item in items"
      :key="item.key"
      class="tab"
      :class="{ active: tab === item.key }"
      @click="emit('change', item.key)"
    >
      <span class="tab-icon">
        <AppIcon :name="item.icon" :size="22" />
        <span v-if="item.key === 'chats' && chat.totalUnread > 0" class="tab-badge">
          {{ chat.totalUnread > 99 ? '99+' : chat.totalUnread }}
        </span>
      </span>
      <span class="tab-label">{{ item.label }}</span>
    </button>
  </nav>
</template>

<style scoped>
.mobile-tabs {
  display: none;
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: calc(56px + env(safe-area-inset-bottom));
  padding-bottom: env(safe-area-inset-bottom);
  background: rgba(250, 250, 250, 0.96);
  backdrop-filter: blur(12px);
  border-top: 1px solid var(--line);
  z-index: 40;
  align-items: stretch;
}

.tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  color: #7b7b7b;
  font-size: 10px;
}

.tab.active {
  color: var(--green);
}

.tab-icon {
  position: relative;
  display: flex;
}

.tab-badge {
  position: absolute;
  top: -5px;
  right: -11px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: #fa5151;
  color: #fff;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1.5px solid #fafafa;
}

@media (max-width: 860px) {
  .mobile-tabs {
    display: flex;
  }
}
</style>
