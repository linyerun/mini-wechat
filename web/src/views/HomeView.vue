<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { Tab } from '@/types'
import { chatSocket } from '@/api/ws'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { errorText } from '@/utils/format'
import NavRail from '@/components/NavRail.vue'
import MobileTabBar from '@/components/MobileTabBar.vue'
import ChatListView from '@/components/ChatListView.vue'
import ContactsView from '@/components/ContactsView.vue'
import ChatWindow from '@/components/ChatWindow.vue'
import NewChatModal from '@/components/NewChatModal.vue'
import AppIcon from '@/components/AppIcon.vue'
import MomentsView from '@/views/MomentsView.vue'
import ProfileView from '@/views/ProfileView.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const ui = useUiStore()

const tab = ref<Tab>('chats')
const mobileChatOpen = ref(false)
const showNewChat = ref(false)
const logoutOpen = ref(false)

const active = computed(() => chat.activeConversation)

let offSocket: (() => void) | null = null

onMounted(async () => {
  offSocket = chatSocket.on((event) => chat.handleEvent(event))
  chatSocket.connect()

  try {
    await Promise.all([chat.loadConversations(), chat.loadContacts()])
    const desktop = window.matchMedia('(min-width: 861px)').matches
    if (desktop && !chat.activeId && chat.conversations.length > 0) {
      await chat.openConversation(chat.conversations[0].id)
    }
  } catch (err) {
    ui.toast(errorText(err), 'error')
  }
})

onBeforeUnmount(() => {
  offSocket?.()
  offSocket = null
  chatSocket.disconnect()
})

watch(
  () => chat.totalUnread,
  (count) => {
    document.title = count > 0 ? `(${count}) 微信 Mini` : '微信 Mini'
  },
  { immediate: true },
)

function switchTab(next: Tab) {
  tab.value = next
  mobileChatOpen.value = false
}

async function openChat(id: number) {
  showNewChat.value = false
  try {
    await chat.openConversation(id)
    mobileChatOpen.value = true
  } catch (err) {
    ui.toast(errorText(err), 'error')
  }
}

function onCreated(id: number) {
  showNewChat.value = false
  void openChat(id)
}

function signOut() {
  logoutOpen.value = false
  chatSocket.disconnect()
  auth.logout()
  chat.reset()
  void router.replace('/login')
}
</script>

<template>
  <div
    class="home"
    :class="{ 'tab-page': tab === 'discover' || tab === 'me', 'mobile-chat-open': mobileChatOpen }"
  >
    <NavRail :tab="tab" @change="switchTab" @logout="logoutOpen = true" />

    <section v-if="tab === 'chats' || tab === 'contacts'" class="list-panel">
      <ChatListView
        v-if="tab === 'chats'"
        :active-id="chat.activeId"
        @open="openChat"
        @new-chat="showNewChat = true"
      />
      <ContactsView v-else @open-chat="openChat" />
    </section>

    <main v-if="tab === 'discover' || tab === 'me' || active" class="main-panel">
      <MomentsView v-if="tab === 'discover'" />
      <ProfileView v-else-if="tab === 'me'" @logout="logoutOpen = true" />
      <ChatWindow
        v-else-if="active"
        :key="active.id"
        :conversation="active"
        @close="mobileChatOpen = false"
      />
    </main>

    <main v-else class="main-panel placeholder-panel">
      <div class="placeholder">
        <div class="placeholder-logo"><AppIcon name="chat" :size="34" /></div>
        <h2>微信 Mini</h2>
        <p>选择一个会话开始聊天，或发起新的对话</p>
        <button class="primary-btn" @click="showNewChat = true">发起聊天</button>
      </div>
    </main>

    <MobileTabBar v-if="!mobileChatOpen" :tab="tab" @change="switchTab" />

    <NewChatModal
      v-if="showNewChat"
      @close="showNewChat = false"
      @created="onCreated"
    />

    <div v-if="logoutOpen" class="modal-mask" @click.self="logoutOpen = false">
      <div class="modal confirm-modal">
        <div class="modal-head">
          <h3>退出登录</h3>
        </div>
        <div class="modal-body">
          <p class="confirm-text">确定要退出当前账号吗？</p>
          <div class="modal-actions">
            <button class="ghost-btn" @click="logoutOpen = false">取消</button>
            <button class="danger-btn" @click="signOut">退出</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home {
  height: 100dvh;
  display: flex;
  overflow: hidden;
  background: var(--bg);
}

.list-panel {
  width: 308px;
  flex: none;
  min-height: 0;
  border-right: 1px solid #dedede;
  overflow: hidden;
}

.main-panel {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #f4f4f4;
}

.placeholder-panel {
  align-items: center;
  justify-content: center;
}

.placeholder {
  text-align: center;
  padding: 30px;
  color: var(--muted);
}

.placeholder-logo {
  width: 78px;
  height: 78px;
  border-radius: 24px;
  margin: 0 auto 20px;
  display: grid;
  place-items: center;
  color: #fff;
  background: linear-gradient(150deg, #0bc464, #07a854);
  box-shadow: 0 16px 38px rgba(7, 193, 96, 0.28);
}

.placeholder h2 {
  margin: 0 0 7px;
  font-size: 20px;
  font-weight: 600;
  color: #333;
}

.placeholder p {
  margin: 0 0 18px;
  font-size: 12.5px;
}

.confirm-modal {
  max-width: 340px;
}

.confirm-text {
  margin: 4px 0 6px;
  font-size: 13.5px;
  color: var(--text-2);
}

@media (max-width: 860px) {
  .list-panel {
    width: 100%;
    border-right: none;
    padding-bottom: calc(56px + env(safe-area-inset-bottom));
  }

  .main-panel {
    display: none;
  }

  .home.tab-page .main-panel {
    display: flex;
    padding-bottom: calc(56px + env(safe-area-inset-bottom));
  }

  .home.mobile-chat-open .list-panel {
    display: none;
  }

  .home.mobile-chat-open .main-panel {
    display: flex;
    padding-bottom: 0;
  }
}
</style>
