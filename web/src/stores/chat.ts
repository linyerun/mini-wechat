import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Conversation, Message, User, WsEvent } from '@/types'
import { api } from '@/api/http'
import { useAuthStore } from './auth'

interface TypingState {
  userId: number
  at: number
}

export const useChatStore = defineStore('chat', () => {
  const auth = useAuthStore()

  const conversations = ref<Conversation[]>([])
  const contacts = ref<User[]>([])
  const activeId = ref<number | null>(null)
  const messages = ref<Record<number, Message[]>>({})
  const hasMore = ref<Record<number, boolean>>({})
  const loadingMessages = ref(false)
  const typing = ref<Record<number, TypingState | undefined>>({})
  const contactsLoaded = ref(false)

  const activeConversation = computed(
    () => conversations.value.find((c) => c.id === activeId.value) ?? null,
  )
  const totalUnread = computed(() =>
    conversations.value.reduce((sum, c) => sum + c.unread, 0),
  )

  function sortConversations() {
    conversations.value.sort((a, b) => {
      const at = a.lastMessage?.createdAt ?? a.createdAt
      const bt = b.lastMessage?.createdAt ?? b.createdAt
      return bt - at
    })
  }

  function upsertConversation(conv: Conversation) {
    const idx = conversations.value.findIndex((c) => c.id === conv.id)
    if (idx >= 0) conversations.value[idx] = conv
    else conversations.value.push(conv)
    sortConversations()
  }

  async function loadConversations() {
    conversations.value = await api.get<Conversation[]>('/conversations')
  }

  async function loadContacts(force = false) {
    if (contactsLoaded.value && !force) return
    contacts.value = await api.get<User[]>('/users')
    contactsLoaded.value = true
  }

  async function loadMessages(id: number) {
    loadingMessages.value = true
    try {
      const list = await api.get<Message[]>(`/conversations/${id}/messages`)
      messages.value[id] = list
      hasMore.value[id] = list.length >= 50
    } finally {
      loadingMessages.value = false
    }
  }

  async function markRead(id: number) {
    const conv = conversations.value.find((c) => c.id === id)
    if (conv) conv.unread = 0
    const list = messages.value[id]
    const lastId = list && list.length > 0 ? list[list.length - 1].id : 0
    try {
      await api.post(`/conversations/${id}/read`, { messageId: lastId })
    } catch {
      // Read receipts are cosmetic; ignore sync failures.
    }
  }

  async function openConversation(id: number) {
    activeId.value = id
    typing.value[id] = undefined
    await loadMessages(id)
    await markRead(id)
  }

  function appendMessage(msg: Message) {
    const list = messages.value[msg.conversationId]
    if (list) {
      if (!list.some((m) => m.id === msg.id)) list.push(msg)
    } else if (activeId.value === msg.conversationId) {
      void loadMessages(msg.conversationId)
    }

    const conv = conversations.value.find((c) => c.id === msg.conversationId)
    if (!conv) return
    conv.lastMessage = msg
    const mine = msg.senderId === auth.user?.id
    if (!mine && activeId.value !== msg.conversationId) conv.unread += 1
    sortConversations()
    if (!mine && activeId.value === msg.conversationId) void markRead(msg.conversationId)
  }

  async function sendMessage(type: 'text' | 'image', content: string) {
    const id = activeId.value
    if (!id) return
    const msg = await api.post<Message>(`/conversations/${id}/messages`, { type, content })
    appendMessage(msg)
  }

  async function loadOlder(id: number): Promise<number> {
    const list = messages.value[id]
    if (!list || list.length === 0) return 0
    const older = await api.get<Message[]>(
      `/conversations/${id}/messages?before=${list[0].id}`,
    )
    if (older.length > 0) messages.value[id] = [...older, ...list]
    hasMore.value[id] = older.length >= 50
    return older.length
  }

  async function startDirect(userId: number): Promise<Conversation> {
    const conv = await api.post<Conversation>('/conversations/direct', { userId })
    upsertConversation(conv)
    activeId.value = conv.id
    await loadMessages(conv.id)
    await markRead(conv.id)
    return conv
  }

  async function createGroup(name: string, memberIds: number[]): Promise<Conversation> {
    const conv = await api.post<Conversation>('/conversations/group', { name, memberIds })
    upsertConversation(conv)
    activeId.value = conv.id
    await loadMessages(conv.id)
    await markRead(conv.id)
    return conv
  }

  function handleTyping(conversationId: number, userId: number) {
    if (userId === auth.user?.id) return
    const at = Date.now()
    typing.value[conversationId] = { userId, at }
    window.setTimeout(() => {
      if (typing.value[conversationId]?.at === at) {
        typing.value[conversationId] = undefined
      }
    }, 3200)
  }

  function handleEvent(event: WsEvent) {
    if (event.type === 'message') {
      appendMessage(event.payload)
    } else if (event.type === 'typing') {
      handleTyping(event.payload.conversationId, event.payload.userId)
    } else if (event.type === 'ready') {
      void refresh()
    }
  }

  async function refresh() {
    try {
      await loadConversations()
      if (!activeId.value) return
      await loadMessages(activeId.value)
      const active = conversations.value.find((c) => c.id === activeId.value)
      if (active && active.unread > 0) await markRead(active.id)
    } catch {
      // The socket reconnects on its own; a failed refresh is not fatal.
    }
  }

  function reset() {
    conversations.value = []
    contacts.value = []
    activeId.value = null
    messages.value = {}
    hasMore.value = {}
    typing.value = {}
    contactsLoaded.value = false
  }

  return {
    conversations,
    contacts,
    activeId,
    messages,
    hasMore,
    loadingMessages,
    typing,
    activeConversation,
    totalUnread,
    loadConversations,
    loadContacts,
    openConversation,
    markRead,
    sendMessage,
    loadOlder,
    startDirect,
    createGroup,
    handleEvent,
    refresh,
    reset,
  }
})
