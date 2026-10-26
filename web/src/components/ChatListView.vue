<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import type { Conversation } from '@/types'
import { conversationAvatar, conversationTitle, formatListTime } from '@/utils/format'
import AppAvatar from './AppAvatar.vue'
import AppIcon from './AppIcon.vue'

defineProps<{ activeId: number | null }>()
const emit = defineEmits<{ (e: 'open', id: number): void; (e: 'new-chat'): void }>()

const auth = useAuthStore()
const chat = useChatStore()
const keyword = ref('')

const list = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return chat.conversations
  return chat.conversations.filter((c) => {
    const title = conversationTitle(c, auth.user?.id)
    const preview = c.lastMessage?.content ?? ''
    return title.toLowerCase().includes(q) || preview.toLowerCase().includes(q)
  })
})

function preview(c: Conversation): string {
  const msg = c.lastMessage
  if (!msg) return '还没有消息'
  const mine = msg.senderId === auth.user?.id
  const body = msg.type === 'image' ? '[图片]' : msg.content
  if (c.type === 'group' && !mine && msg.sender) return `${msg.sender.nickname}：${body}`
  return mine ? `我：${body}` : body
}

function avatarOf(c: Conversation) {
  return conversationAvatar(c, auth.user?.id)
}
</script>

<template>
  <div class="chat-list-panel">
    <header class="panel-header">
      <span class="panel-title">微信</span>
      <button class="icon-btn" title="发起聊天" @click="emit('new-chat')">
        <AppIcon name="plus" :size="20" />
      </button>
    </header>

    <div class="search-box">
      <AppIcon name="search" :size="15" />
      <input v-model="keyword" placeholder="搜索会话" />
    </div>

    <div class="conversation-list">
      <button
        v-for="c in list"
        :key="c.id"
        class="conversation"
        :class="{ active: c.id === activeId }"
        @click="emit('open', c.id)"
      >
        <AppAvatar
          :name="avatarOf(c).name"
          :src="avatarOf(c).src"
          :square="c.type === 'group'"
          :size="46"
        />
        <div class="conv-main">
          <div class="conv-top">
            <span class="conv-name">{{ conversationTitle(c, auth.user?.id) }}</span>
            <span class="conv-time">{{
              formatListTime(c.lastMessage?.createdAt ?? c.createdAt)
            }}</span>
          </div>
          <div class="conv-bottom">
            <span class="conv-preview">{{ preview(c) }}</span>
            <span v-if="c.unread > 0" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
          </div>
        </div>
      </button>

      <div v-if="!list.length" class="empty-state">
        <div class="empty-icon"><AppIcon name="chat" :size="28" /></div>
        <p>{{ keyword ? '没有找到相关会话' : '还没有会话，开始聊天吧' }}</p>
        <button v-if="!keyword" class="primary-btn" @click="emit('new-chat')">发起聊天</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-list-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--list-bg);
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.conversation {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 10px;
  border-radius: 8px;
  text-align: left;
  transition: background 0.12s ease;
}

.conversation:hover {
  background: rgba(0, 0, 0, 0.045);
}

.conversation.active {
  background: #d7d7d7;
}

.conv-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.conv-top,
.conv-bottom {
  display: flex;
  align-items: center;
  gap: 8px;
}

.conv-name {
  flex: 1;
  min-width: 0;
  font-size: 14.5px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-time {
  flex: none;
  font-size: 10.5px;
  color: var(--muted);
}

.conv-preview {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.empty-state {
  padding-top: 60px;
  height: auto;
}

.empty-state p {
  margin: 0;
}

.empty-state .primary-btn {
  margin-top: 4px;
}
</style>
