<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import type { Moment, User } from '@/types'
import { api } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { errorText } from '@/utils/format'
import AppAvatar from '@/components/AppAvatar.vue'
import AppIcon from '@/components/AppIcon.vue'

const emit = defineEmits<{ (e: 'logout'): void }>()

const auth = useAuthStore()
const chat = useChatStore()
const ui = useUiStore()

const editing = ref(false)
const saving = ref(false)
const uploading = ref(false)
const momentCount = ref<number | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const form = reactive({ nickname: '', signature: '', avatar: '' })

onMounted(async () => {
  chat.loadContacts().catch(() => undefined)
  try {
    const moments = await api.get<Moment[]>('/moments')
    momentCount.value = moments.length
  } catch {
    momentCount.value = null
  }
})

function openEdit() {
  form.nickname = auth.user?.nickname ?? ''
  form.signature = auth.user?.signature ?? ''
  form.avatar = auth.user?.avatar ?? ''
  editing.value = true
}

function pickAvatar() {
  fileInput.value?.click()
}

async function onAvatarChange(event: Event) {
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
    form.avatar = await api.upload(file)
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    uploading.value = false
  }
}

async function save() {
  const nickname = form.nickname.trim()
  if (!nickname) {
    ui.toast('昵称不能为空', 'error')
    return
  }
  saving.value = true
  try {
    const res = await api.patch<{ user: User }>('/me', {
      nickname,
      signature: form.signature.trim(),
      avatar: form.avatar,
    })
    auth.updateUser(res.user)
    editing.value = false
    ui.toast('资料已更新')
    await chat.refresh()
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="profile-page">
    <header class="page-head">
      <h2>我</h2>
    </header>

    <div class="profile-scroll">
      <div class="profile-inner">
        <section class="profile-card">
          <AppAvatar
            :name="auth.user?.nickname ?? '我'"
            :src="auth.user?.avatar"
            :size="84"
          />
          <div class="profile-info">
            <h3>{{ auth.user?.nickname }}</h3>
            <p class="handle">@{{ auth.user?.username }}</p>
            <p class="signature">{{ auth.user?.signature || '这个人很懒，什么都没留下' }}</p>
          </div>
          <button class="ghost-btn edit-btn" @click="openEdit">
            <AppIcon name="edit" :size="15" />
            编辑资料
          </button>
        </section>

        <section class="stats-card">
          <div class="stat">
            <b>{{ chat.conversations.length }}</b>
            <span>会话</span>
          </div>
          <div class="stat-divider" />
          <div class="stat">
            <b>{{ chat.contacts.length }}</b>
            <span>联系人</span>
          </div>
          <div class="stat-divider" />
          <div class="stat">
            <b>{{ momentCount ?? '—' }}</b>
            <span>朋友圈</span>
          </div>
        </section>

        <section class="about-card">
          <h4>关于 Mini 微信</h4>
          <p>一个使用 Go + Vue 3 搭建的全栈聊天演示，支持单聊、群聊、朋友圈与实时消息。</p>
          <ul>
            <li>Go · Gin · SQLite · WebSocket</li>
            <li>Vue 3 · TypeScript · Vite · Pinia</li>
            <li>JWT 鉴权 · 本地图片上传</li>
          </ul>
        </section>

        <button class="danger-btn logout-btn" @click="emit('logout')">
          <AppIcon name="logout" :size="16" />
          退出登录
        </button>
        <p class="version">Mini WeChat · v0.1.0</p>
      </div>
    </div>

    <div v-if="editing" class="modal-mask" @click.self="editing = false">
      <div class="modal">
        <div class="modal-head">
          <h3>编辑资料</h3>
          <button class="icon-btn" @click="editing = false">
            <AppIcon name="close" :size="18" />
          </button>
        </div>
        <div class="modal-body">
          <div class="avatar-edit">
            <AppAvatar :name="form.nickname || '我'" :src="form.avatar" :size="72" />
            <button class="ghost-btn" :disabled="uploading" @click="pickAvatar">
              <AppIcon name="camera" :size="15" />
              {{ uploading ? '上传中…' : '更换头像' }}
            </button>
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              hidden
              @change="onAvatarChange"
            />
          </div>
          <label class="field">
            <span>昵称</span>
            <input v-model="form.nickname" maxlength="20" placeholder="你的昵称" />
          </label>
          <label class="field">
            <span>个性签名</span>
            <textarea v-model="form.signature" maxlength="60" rows="3" placeholder="介绍一下自己…" />
          </label>
          <div class="modal-actions">
            <button class="ghost-btn" @click="editing = false">取消</button>
            <button class="primary-btn" :disabled="saving || uploading" @click="save">
              {{ saving ? '保存中…' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f6f5;
}

.page-head {
  flex: none;
  height: 62px;
  padding: 0 26px;
  display: flex;
  align-items: center;
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--line);
}

.page-head h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.profile-scroll {
  flex: 1;
  overflow-y: auto;
}

.profile-inner {
  width: min(640px, 100%);
  margin: 0 auto;
  padding: 24px 22px 60px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.profile-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 26px 24px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(20, 40, 30, 0.05);
}

.profile-info {
  flex: 1;
  min-width: 0;
}

.profile-info h3 {
  margin: 0;
  font-size: 21px;
  font-weight: 600;
}

.handle {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--muted);
}

.signature {
  margin: 9px 0 0;
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
}

.edit-btn {
  flex: none;
  padding: 8px 13px;
  font-size: 12.5px;
}

.stats-card {
  display: flex;
  align-items: center;
  padding: 18px 10px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(20, 40, 30, 0.05);
}

.stat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat b {
  font-size: 20px;
  font-weight: 600;
  color: var(--text);
}

.stat span {
  font-size: 11.5px;
  color: var(--muted);
}

.stat-divider {
  width: 1px;
  height: 26px;
  background: var(--line);
}

.about-card {
  padding: 20px 22px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(20, 40, 30, 0.05);
}

.about-card h4 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.about-card p {
  margin: 0 0 10px;
  font-size: 12.5px;
  color: var(--text-2);
  line-height: 1.65;
}

.about-card ul {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.about-card li {
  font-size: 12px;
  color: var(--muted);
  padding-left: 14px;
  position: relative;
}

.about-card li::before {
  content: '';
  position: absolute;
  left: 2px;
  top: 7px;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #c9c9c9;
}

.logout-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  height: 46px;
  border-radius: 12px;
  font-size: 13.5px;
}

.version {
  margin: 2px 0 0;
  text-align: center;
  font-size: 10.5px;
  color: #bcbcbc;
}

.avatar-edit {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 0 2px;
}

.avatar-edit .ghost-btn {
  padding: 8px 13px;
  font-size: 12.5px;
}

@media (max-width: 860px) {
  .page-head {
    padding: 0 16px;
    height: 56px;
  }

  .profile-inner {
    padding: 16px 12px 40px;
  }

  .profile-card {
    flex-wrap: wrap;
    padding: 20px;
    gap: 14px;
  }

  .profile-info {
    flex-basis: calc(100% - 104px);
  }

  .edit-btn {
    width: 100%;
    height: 38px;
  }
}
</style>
