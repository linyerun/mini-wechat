<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { Conversation, Message } from '@/types'
import { api } from '@/api/http'
import { chatSocket } from '@/api/ws'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import {
  conversationAvatar,
  conversationTitle,
  errorText,
  formatDayLabel,
  formatMsgTime,
} from '@/utils/format'
import AppAvatar from './AppAvatar.vue'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ conversation: Conversation }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const auth = useAuthStore()
const chat = useChatStore()
const ui = useUiStore()

const draft = ref('')
const sending = ref(false)
const uploading = ref(false)
const emojiOpen = ref(false)
const lightbox = ref('')
const loadingOlder = ref(false)
const stick = ref(true)

const scrollEl = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLTextAreaElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const selfId = computed(() => auth.user?.id ?? 0)
const other = computed(() => props.conversation.members.find((m) => m.id !== selfId.value))
const title = computed(() => conversationTitle(props.conversation, selfId.value))
const avatar = computed(() => conversationAvatar(props.conversation, selfId.value))
const list = computed(() => chat.messages[props.conversation.id] ?? [])

const typingUser = computed(() => {
  const state = chat.typing[props.conversation.id]
  if (!state) return null
  return props.conversation.members.find((m) => m.id === state.userId) ?? null
})

const subtitle = computed(() => {
  if (typingUser.value) return `${typingUser.value.nickname} 正在输入…`
  if (props.conversation.type === 'group') return `${props.conversation.members.length} 位成员`
  return other.value?.signature || (other.value ? `@${other.value.username}` : '')
})

type Row =
  | { kind: 'date'; key: string; label: string }
  | { kind: 'msg'; key: string; message: Message }

const rows = computed<Row[]>(() => {
  const out: Row[] = []
  let lastDay = ''
  for (const message of list.value) {
    const d = new Date(message.createdAt)
    const dayKey = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
    if (dayKey !== lastDay) {
      out.push({ kind: 'date', key: `date-${dayKey}`, label: formatDayLabel(message.createdAt) })
      lastDay = dayKey
    }
    out.push({ kind: 'msg', key: `msg-${message.id}`, message })
  }
  return out
})

const emojis = [
  '😀', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '😍', '😘', '😜', '🤪',
  '😎', '🤗', '🤔', '😐', '😴', '😭', '😤', '😡', '🥳', '🤩', '👍', '👌',
  '🙏', '💪', '👏', '🙌', '🤝', '❤️', '💚', '💙', '💛', '💔', '✨', '🔥',
  '🎉', '🎂', '☕', '🌹', '🌙', '☀️', '🌈', '🍺', '🍜', '🚀', '⚽️', '🐶',
]

function isSelf(message: Message): boolean {
  return message.senderId === selfId.value
}

function showSender(message: Message): boolean {
  return props.conversation.type === 'group' && !isSelf(message)
}

function senderName(message: Message): string {
  return message.sender?.nickname ?? '未知用户'
}

function onScroll() {
  const el = scrollEl.value
  if (!el) return
  stick.value = el.scrollHeight - el.scrollTop - el.clientHeight < 120
}

function scrollToBottom() {
  const el = scrollEl.value
  if (el) el.scrollTop = el.scrollHeight
}

watch(
  () => list.value.length,
  async (next, previous) => {
    const grew = next > (previous ?? 0)
    const last = list.value[list.value.length - 1]
    const mine = last ? last.senderId === selfId.value : false
    await nextTick()
    if (grew && (stick.value || mine)) scrollToBottom()
  },
)

onMounted(async () => {
  if (!chat.messages[props.conversation.id]) {
    try {
      await chat.openConversation(props.conversation.id)
    } catch (err) {
      ui.toast(errorText(err), 'error')
    }
  }
  await nextTick()
  scrollToBottom()
})

let lastTypingAt = 0

function onInput() {
  const el = inputEl.value
  if (el) {
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 120)}px`
  }
  const now = Date.now()
  if (now - lastTypingAt > 2000) {
    lastTypingAt = now
    chatSocket.send({ type: 'typing', conversationId: props.conversation.id })
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
    event.preventDefault()
    void sendText()
  }
}

async function sendText() {
  if (sending.value) return
  const text = draft.value.trim()
  if (!text) return
  sending.value = true
  try {
    await chat.sendMessage('text', text)
    draft.value = ''
    emojiOpen.value = false
    stick.value = true
    if (inputEl.value) inputEl.value.style.height = 'auto'
    await nextTick()
    scrollToBottom()
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    sending.value = false
  }
}

function insertEmoji(emoji: string) {
  draft.value += emoji
  inputEl.value?.focus()
}

function pickImage() {
  fileInput.value?.click()
}

async function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (file.size > 8 * 1024 * 1024) {
    ui.toast('图片不能超过 8MB', 'error')
    return
  }
  uploading.value = true
  try {
    const url = await api.upload(file)
    await chat.sendMessage('image', url)
    stick.value = true
    await nextTick()
    scrollToBottom()
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    uploading.value = false
  }
}

async function loadOlder() {
  if (loadingOlder.value) return
  loadingOlder.value = true
  const el = scrollEl.value
  const beforeHeight = el?.scrollHeight ?? 0
  const beforeTop = el?.scrollTop ?? 0
  try {
    const count = await chat.loadOlder(props.conversation.id)
    if (count > 0) {
      await nextTick()
      if (el) el.scrollTop = el.scrollHeight - beforeHeight + beforeTop
    }
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    loadingOlder.value = false
  }
}
</script>

<template>
  <section class="chat-window">
    <header class="chat-head">
      <button class="back-btn" title="返回" @click="emit('close')">
        <AppIcon name="back" :size="20" />
      </button>
      <div class="head-main">
        <div class="head-title">
          {{ title }}
          <span v-if="conversation.type === 'group'" class="head-count">
            {{ conversation.members.length }}
          </span>
        </div>
        <div class="head-sub" :class="{ typing: typingUser }">{{ subtitle }}</div>
      </div>
      <AppAvatar
        :name="avatar.name"
        :src="avatar.src"
        :square="conversation.type === 'group'"
        :size="34"
      />
    </header>

    <div ref="scrollEl" class="messages" @scroll="onScroll">
      <button
        v-if="chat.hasMore[conversation.id]"
        class="load-more"
        :disabled="loadingOlder"
        @click="loadOlder"
      >
        {{ loadingOlder ? '加载中…' : '查看更早的消息' }}
      </button>

      <template v-for="row in rows" :key="row.key">
        <div v-if="row.kind === 'msg'" class="msg-row" :class="{ self: isSelf(row.message) }">
          <AppAvatar
            v-if="!isSelf(row.message)"
            :name="row.message.sender?.nickname ?? '?'"
            :src="row.message.sender?.avatar"
            :size="36"
          />
          <div class="msg-body">
            <div v-if="showSender(row.message)" class="msg-sender">
              {{ senderName(row.message) }}
            </div>
            <div
              v-if="row.message.type === 'text'"
              class="bubble"
              :class="{ self: isSelf(row.message) }"
            >
              {{ row.message.content }}
            </div>
            <button
              v-else
              class="image-wrap"
              title="查看大图"
              @click="lightbox = row.message.content"
            >
              <img :src="row.message.content" alt="图片消息" />
            </button>
            <span v-if="isSelf(row.message)" class="msg-time">
              {{ formatMsgTime(row.message.createdAt) }}
            </span>
          </div>
          <AppAvatar
            v-if="isSelf(row.message)"
            :name="auth.user?.nickname ?? '我'"
            :src="auth.user?.avatar"
            :size="36"
          />
        </div>
        <div v-else class="date-divider"><span>{{ row.label }}</span></div>
      </template>

      <div v-if="!rows.length" class="empty-state chat-empty">
        <div class="empty-icon"><AppIcon name="chat" :size="26" /></div>
        <p>和 {{ title }} 打个招呼吧</p>
      </div>
    </div>

    <div class="composer">
      <div v-if="emojiOpen" class="emoji-pop">
        <button v-for="emoji in emojis" :key="emoji" @click="insertEmoji(emoji)">
          {{ emoji }}
        </button>
      </div>

      <div class="composer-tools">
        <button class="tool" title="表情" @click="emojiOpen = !emojiOpen">
          <AppIcon name="emoji" :size="21" />
        </button>
        <button class="tool" title="发送图片" :disabled="uploading" @click="pickImage">
          <AppIcon name="image" :size="21" />
        </button>
        <span v-if="uploading" class="uploading">图片上传中…</span>
      </div>

      <div class="composer-main">
        <textarea
          ref="inputEl"
          v-model="draft"
          rows="1"
          placeholder="输入消息，Enter 发送，Shift + Enter 换行"
          @input="onInput"
          @keydown="onKeydown"
        />
        <button class="send-btn" :disabled="!draft.trim() || sending" @click="sendText">
          发送
        </button>
      </div>

      <input ref="fileInput" type="file" accept="image/*" hidden @change="onFileChange" />
    </div>

    <div v-if="lightbox" class="lightbox" @click="lightbox = ''">
      <img :src="lightbox" alt="图片预览" />
    </div>
  </section>
</template>

<style scoped>
.chat-window {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f4f4f4;
  min-width: 0;
}

.chat-head {
  height: 64px;
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 20px;
  background: #fff;
  border-bottom: 1px solid var(--line);
}

.back-btn {
  display: none;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  align-items: center;
  justify-content: center;
  color: var(--text);
  margin-left: -6px;
}

.back-btn:hover {
  background: #f2f2f2;
}

.head-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.head-title {
  font-size: 15.5px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 7px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.head-count {
  font-size: 10.5px;
  font-weight: 500;
  color: var(--muted);
  background: #f1f1f1;
  border-radius: 9px;
  padding: 1px 7px;
}

.head-sub {
  font-size: 11.5px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.head-sub.typing {
  color: var(--green);
}

.messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 18px 26px 14px;
}

.load-more {
  display: block;
  margin: 2px auto 16px;
  font-size: 11.5px;
  color: var(--green);
  padding: 4px 10px;
  border-radius: 12px;
  background: rgba(7, 193, 96, 0.08);
}

.load-more:disabled {
  color: var(--muted);
}

.date-divider {
  display: flex;
  justify-content: center;
  margin: 16px 0 18px;
}

.date-divider span {
  font-size: 11px;
  color: #a8a8a8;
  background: rgba(0, 0, 0, 0.035);
  padding: 3px 9px;
  border-radius: 9px;
}

.msg-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 18px;
  animation: fade-in 0.18s ease both;
}

.msg-row.self {
  flex-direction: row-reverse;
}

.msg-body {
  max-width: min(62%, 520px);
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
}

.msg-row.self .msg-body {
  align-items: flex-end;
}

.msg-sender {
  font-size: 11.5px;
  color: var(--muted);
  margin: 2px 0 5px 2px;
}

.bubble {
  position: relative;
  padding: 9px 12px;
  border-radius: 3px 9px 9px 9px;
  background: var(--bubble-other);
  font-size: 14px;
  line-height: 1.55;
  color: var(--text);
  word-break: break-word;
  white-space: pre-wrap;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.045);
}

.bubble.self {
  background: var(--bubble-self);
  border-radius: 9px 3px 9px 9px;
}

.image-wrap {
  border-radius: 9px;
  overflow: hidden;
  display: block;
  max-width: 220px;
  background: #e3e3e3;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

.image-wrap img {
  display: block;
  width: 100%;
  max-height: 280px;
  object-fit: cover;
}

.msg-time {
  margin-top: 5px;
  font-size: 10px;
  color: #b3b3b3;
}

.chat-empty {
  height: 100%;
  padding: 0;
}

.composer {
  position: relative;
  flex: none;
  padding: 9px 16px 14px;
  background: #fff;
  border-top: 1px solid var(--line);
}

.composer-tools {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-bottom: 6px;
}

.tool {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #4a4a4a;
}

.tool:hover {
  background: #f2f2f2;
  color: var(--green);
}

.uploading {
  margin-left: 6px;
  font-size: 11px;
  color: var(--muted);
}

.composer-main {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.composer-main textarea {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  resize: none;
  background: #f4f4f4;
  border-radius: 8px;
  padding: 9px 11px;
  min-height: 38px;
  max-height: 120px;
  line-height: 1.5;
  font-size: 14px;
}

.composer-main textarea::placeholder {
  color: #b0b0b0;
  font-size: 12.5px;
}

.send-btn {
  flex: none;
  height: 34px;
  padding: 0 17px;
  border-radius: 7px;
  background: var(--green);
  color: #fff;
  font-size: 13px;
  margin-bottom: 2px;
  transition: background 0.15s ease;
}

.send-btn:hover:not(:disabled) {
  background: var(--green-hover);
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.emoji-pop {
  position: absolute;
  left: 14px;
  bottom: calc(100% - 2px);
  width: min(340px, calc(100vw - 40px));
  max-height: 186px;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 2px;
  padding: 10px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid var(--line);
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.14);
  z-index: 20;
}

.emoji-pop button {
  height: 34px;
  border-radius: 6px;
  font-size: 19px;
  line-height: 1;
}

.emoji-pop button:hover {
  background: #f3f3f3;
}

.lightbox {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.86);
  display: grid;
  place-items: center;
  z-index: 150;
  cursor: zoom-out;
}

.lightbox img {
  max-width: 90vw;
  max-height: 88vh;
  border-radius: 6px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
}

@media (max-width: 860px) {
  .chat-head {
    padding: 0 12px;
  }

  .back-btn {
    display: inline-flex;
  }

  .messages {
    padding: 14px 12px 10px;
  }

  .msg-body {
    max-width: min(72%, 460px);
  }

  .composer {
    padding: 8px 10px calc(10px + env(safe-area-inset-bottom));
  }

  .composer-main textarea::placeholder {
    font-size: 12px;
  }
}
</style>
