<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { conversationAvatar, conversationTitle, errorText } from '@/utils/format'
import AppAvatar from './AppAvatar.vue'
import AppIcon from './AppIcon.vue'

const emit = defineEmits<{ (e: 'open-chat', id: number): void }>()

const auth = useAuthStore()
const chat = useChatStore()
const ui = useUiStore()
const keyword = ref('')
const busy = ref(false)

onMounted(() => {
  chat.loadContacts().catch((err) => ui.toast(errorText(err), 'error'))
})

const contacts = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return chat.contacts
  return chat.contacts.filter(
    (u) => u.nickname.toLowerCase().includes(q) || u.username.toLowerCase().includes(q),
  )
})

const groups = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return chat.conversations.filter(
    (c) => c.type === 'group' && (!q || c.name.toLowerCase().includes(q)),
  )
})

async function start(userId: number) {
  if (busy.value) return
  busy.value = true
  try {
    const conv = await chat.startDirect(userId)
    emit('open-chat', conv.id)
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="contacts-panel">
    <header class="panel-header">
      <span class="panel-title">通讯录</span>
      <span class="contact-count">{{ chat.contacts.length }} 位联系人</span>
    </header>

    <div class="search-box">
      <AppIcon name="search" :size="15" />
      <input v-model="keyword" placeholder="搜索联系人 / 群聊" />
    </div>

    <div class="contacts-scroll">
      <section v-if="groups.length" class="contact-section">
        <h4>群聊</h4>
        <button
          v-for="group in groups"
          :key="group.id"
          class="contact-row"
          @click="emit('open-chat', group.id)"
        >
          <AppAvatar
            :name="conversationAvatar(group, auth.user?.id).name"
            :src="conversationAvatar(group, auth.user?.id).src"
            square
            :size="42"
          />
          <div class="contact-main">
            <span class="contact-name">{{ conversationTitle(group, auth.user?.id) }}</span>
            <span class="contact-sub">{{ group.members.length }} 位成员</span>
          </div>
          <AppIcon name="chevron-right" :size="16" class="chev" />
        </button>
      </section>

      <section class="contact-section">
        <h4>联系人</h4>
        <button
          v-for="user in contacts"
          :key="user.id"
          class="contact-row"
          :disabled="busy"
          @click="start(user.id)"
        >
          <AppAvatar :name="user.nickname" :src="user.avatar" :size="42" />
          <div class="contact-main">
            <span class="contact-name">{{ user.nickname }}</span>
            <span class="contact-sub">{{ user.signature || `@${user.username}` }}</span>
          </div>
          <span class="contact-action">发消息</span>
        </button>
        <div v-if="!contacts.length" class="empty-state">
          <p>{{ keyword ? '没有找到相关联系人' : '暂无联系人' }}</p>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.contacts-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--list-bg);
}

.contact-count {
  font-size: 11px;
  color: var(--muted);
}

.contacts-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 14px;
}

.contact-section h4 {
  margin: 10px 10px 6px;
  font-size: 11px;
  font-weight: 500;
  color: var(--muted);
  letter-spacing: 0.6px;
}

.contact-row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 9px 10px;
  border-radius: 8px;
  text-align: left;
  transition: background 0.12s ease;
}

.contact-row:hover {
  background: rgba(0, 0, 0, 0.045);
}

.contact-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.contact-name {
  font-size: 14px;
  font-weight: 500;
}

.contact-sub {
  font-size: 12px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.contact-action {
  font-size: 11px;
  color: var(--green);
  opacity: 0;
  transition: opacity 0.15s ease;
}

.contact-row:hover .contact-action {
  opacity: 1;
}

.chev {
  color: #bdbdbd;
}
</style>
