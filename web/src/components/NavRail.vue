<script setup lang="ts">
import { ref } from 'vue'
import type { Tab } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import AppAvatar from './AppAvatar.vue'
import AppIcon from './AppIcon.vue'

defineProps<{ tab: Tab }>()
const emit = defineEmits<{ (e: 'change', tab: Tab): void; (e: 'logout'): void }>()

const auth = useAuthStore()
const chat = useChatStore()
const menuOpen = ref(false)

function selectTab(next: Tab) {
  menuOpen.value = false
  emit('change', next)
}

function requestLogout() {
  menuOpen.value = false
  emit('logout')
}

const items: { key: Tab; icon: string; label: string }[] = [
  { key: 'chats', icon: 'chat', label: '微信' },
  { key: 'contacts', icon: 'contacts', label: '通讯录' },
  { key: 'discover', icon: 'discover', label: '发现' },
  { key: 'me', icon: 'me', label: '我' },
]
</script>

<template>
  <aside class="nav-rail">
    <div class="nav-profile-wrap">
      <button
        class="nav-profile"
        :class="{ active: menuOpen }"
        :title="auth.user?.nickname"
        @click="menuOpen = !menuOpen"
      >
        <AppAvatar :name="auth.user?.nickname ?? '我'" :src="auth.user?.avatar" :size="38" />
      </button>

      <template v-if="menuOpen">
        <div class="menu-backdrop" @click="menuOpen = false" />
        <div class="avatar-menu">
          <p class="menu-name">{{ auth.user?.nickname }}</p>
          <button class="menu-item" @click="selectTab('me')">
            <AppIcon name="me" :size="15" />
            我的资料
          </button>
          <button class="menu-item menu-danger" @click="requestLogout">
            <AppIcon name="logout" :size="15" />
            退出登录
          </button>
        </div>
      </template>
    </div>
    <nav class="nav-list">
      <button
        v-for="item in items"
        :key="item.key"
        class="nav-item"
        :class="{ active: tab === item.key }"
        :title="item.label"
        @click="emit('change', item.key)"
      >
        <span class="nav-icon">
          <AppIcon :name="item.icon" :size="23" />
          <span v-if="item.key === 'chats' && chat.totalUnread > 0" class="nav-badge">
            {{ chat.totalUnread > 99 ? '99+' : chat.totalUnread }}
          </span>
        </span>
        <span class="nav-label">{{ item.label }}</span>
      </button>
    </nav>
    <div class="nav-spacer" />
    <div class="nav-version">MINI</div>
  </aside>
</template>

<style scoped>
.nav-rail {
  width: 74px;
  flex: none;
  background: var(--rail-bg);
  color: #9a9a9a;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 22px 0 14px;
  gap: 10px;
}

.nav-profile-wrap {
  position: relative;
}

.nav-profile {
  padding: 4px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.nav-profile:hover,
.nav-profile.active {
  transform: scale(1.05);
}

.menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 50;
}

.avatar-menu {
  position: absolute;
  top: -8px;
  left: calc(100% + 14px);
  z-index: 51;
  width: 164px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 10px;
  box-shadow: var(--shadow);
}

.menu-name {
  margin: 4px 8px 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 8px;
  border-radius: 7px;
  font-size: 13px;
  color: #333;
  text-align: left;
  transition: background 0.15s ease;
}

.menu-item:hover {
  background: #f2f2f2;
}

.menu-danger {
  color: var(--danger);
}

.menu-danger:hover {
  background: #fff1f0;
}

.nav-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 14px;
  width: 100%;
  align-items: center;
}

.nav-item {
  width: 58px;
  padding: 9px 0 8px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  color: #9a9a9a;
  transition: background 0.15s ease, color 0.15s ease;
}

.nav-item:hover {
  color: #d7d7d7;
  background: var(--rail-hover);
}

.nav-item.active {
  color: var(--green);
}

.nav-icon {
  position: relative;
  display: flex;
}

.nav-badge {
  position: absolute;
  top: -5px;
  right: -10px;
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
  border: 1.5px solid var(--rail-bg);
}

.nav-label {
  font-size: 10px;
  letter-spacing: 0.3px;
}

.nav-spacer {
  flex: 1;
}

.nav-version {
  font-size: 9px;
  letter-spacing: 1.4px;
  color: #5c5c5c;
}

@media (max-width: 860px) {
  .nav-rail {
    display: none;
  }
}
</style>
