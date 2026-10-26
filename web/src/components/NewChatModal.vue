<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { errorText } from '@/utils/format'
import AppAvatar from './AppAvatar.vue'
import AppIcon from './AppIcon.vue'

const emit = defineEmits<{ (e: 'close'): void; (e: 'created', id: number): void }>()

const chat = useChatStore()
const ui = useUiStore()

const mode = ref<'direct' | 'group'>('direct')
const keyword = ref('')
const groupName = ref('')
const selected = ref<number[]>([])
const busy = ref(false)

onMounted(() => {
  chat.loadContacts().catch((err) => ui.toast(errorText(err), 'error'))
})

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return chat.contacts
  return chat.contacts.filter(
    (u) => u.nickname.toLowerCase().includes(q) || u.username.toLowerCase().includes(q),
  )
})

function toggle(id: number) {
  if (selected.value.includes(id)) {
    selected.value = selected.value.filter((v) => v !== id)
  } else {
    selected.value = [...selected.value, id]
  }
}

async function openDirect(userId: number) {
  if (busy.value) return
  busy.value = true
  try {
    const conv = await chat.startDirect(userId)
    emit('created', conv.id)
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    busy.value = false
  }
}

async function createGroup() {
  if (busy.value) return
  const name = groupName.value.trim()
  if (!name) {
    ui.toast('请输入群名称', 'error')
    return
  }
  if (!selected.value.length) {
    ui.toast('请至少选择一位成员', 'error')
    return
  }
  busy.value = true
  try {
    const conv = await chat.createGroup(name, selected.value)
    emit('created', conv.id)
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="modal-mask" @click.self="emit('close')">
    <div class="modal new-chat-modal">
      <div class="modal-head">
        <h3>发起聊天</h3>
        <button class="icon-btn" @click="emit('close')">
          <AppIcon name="close" :size="18" />
        </button>
      </div>

      <div class="mode-tabs">
        <button :class="{ active: mode === 'direct' }" @click="mode = 'direct'">单聊</button>
        <button :class="{ active: mode === 'group' }" @click="mode = 'group'">群聊</button>
      </div>

      <div class="search-box">
        <AppIcon name="search" :size="15" />
        <input v-model="keyword" placeholder="搜索联系人" />
      </div>

      <div class="modal-body pick-body">
        <label v-if="mode === 'group'" class="field group-name">
          <span>群名称</span>
          <input v-model="groupName" placeholder="给群聊起个名字" maxlength="30" />
        </label>

        <p v-if="mode === 'group'" class="pick-label">
          选择成员 <span>已选 {{ selected.length }} 人</span>
        </p>

        <div class="pick-list">
          <button
            v-for="user in filtered"
            :key="user.id"
            class="pick-row"
            :disabled="busy"
            @click="mode === 'direct' ? openDirect(user.id) : toggle(user.id)"
          >
            <AppAvatar :name="user.nickname" :src="user.avatar" :size="38" />
            <div class="pick-main">
              <span class="pick-name">{{ user.nickname }}</span>
              <span class="pick-sub">{{ user.signature || `@${user.username}` }}</span>
            </div>
            <span v-if="mode === 'group'" class="check" :class="{ on: selected.includes(user.id) }">
              <AppIcon v-if="selected.includes(user.id)" name="check" :size="13" />
            </span>
            <AppIcon v-else name="chevron-right" :size="16" class="chev" />
          </button>
          <div v-if="!filtered.length" class="empty-state">
            <p>没有找到联系人</p>
          </div>
        </div>
      </div>

      <div v-if="mode === 'group'" class="group-actions">
        <button
          class="primary-btn"
          :disabled="busy || !groupName.trim() || !selected.length"
          @click="createGroup"
        >
          {{ busy ? '创建中…' : `创建群聊（${selected.length + 1} 人）` }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.new-chat-modal {
  max-width: 440px;
}

.mode-tabs {
  display: flex;
  gap: 4px;
  background: #f3f3f3;
  border-radius: 10px;
  padding: 4px;
  margin: 4px 20px 12px;
}

.mode-tabs button {
  flex: 1;
  padding: 7px 0;
  font-size: 13px;
  color: var(--text-2);
  border-radius: 7px;
}

.mode-tabs button.active {
  background: #fff;
  color: var(--text);
  font-weight: 500;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.09);
}

.pick-body {
  padding-top: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.group-name {
  margin-top: 4px;
}

.pick-label {
  display: flex;
  justify-content: space-between;
  margin: 14px 2px 6px;
  font-size: 11.5px;
  color: var(--muted);
}

.pick-list {
  overflow-y: auto;
  max-height: 320px;
  margin: 0 -6px;
  padding: 0 6px 4px;
}

.pick-row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border-radius: 8px;
  text-align: left;
}

.pick-row:hover:not(:disabled) {
  background: #f6f6f6;
}

.pick-row:disabled {
  opacity: 0.6;
  cursor: default;
}

.pick-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.pick-name {
  font-size: 13.5px;
  font-weight: 500;
}

.pick-sub {
  font-size: 11.5px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chev {
  color: #c4c4c4;
}

.check {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1.5px solid #d6d6d6;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex: none;
}

.check.on {
  background: var(--green);
  border-color: var(--green);
}

.group-actions {
  padding: 0 20px 20px;
}

.group-actions .primary-btn {
  width: 100%;
  height: 42px;
}
</style>
