<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Moment, MomentComment, User } from '@/types'
import { api } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorText, formatRelative } from '@/utils/format'
import AppAvatar from '@/components/AppAvatar.vue'
import AppIcon from '@/components/AppIcon.vue'

const auth = useAuthStore()
const ui = useUiStore()

const moments = ref<Moment[]>([])
const loading = ref(false)
const draft = ref('')
const images = ref<string[]>([])
const composerOpen = ref(false)
const posting = ref(false)
const uploading = ref(false)
const commenting = ref<number | null>(null)
const commentDrafts = ref<Record<number, string>>({})
const busyLike = ref<number | null>(null)
const preview = ref('')

const fileInput = ref<HTMLInputElement | null>(null)

async function load() {
  if (loading.value) return
  loading.value = true
  try {
    moments.value = await api.get<Moment[]>('/moments')
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function chooseImages() {
  fileInput.value?.click()
}

async function onFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (!files.length) return
  uploading.value = true
  try {
    for (const file of files) {
      if (images.value.length >= 9) break
      if (file.size > 8 * 1024 * 1024) {
        ui.toast(`${file.name} 超过 8MB`, 'error')
        continue
      }
      const url = await api.upload(file)
      images.value.push(url)
    }
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    uploading.value = false
  }
}

async function publish() {
  const content = draft.value.trim()
  if (posting.value || (!content && !images.value.length)) return
  posting.value = true
  try {
    const moment = await api.post<Moment>('/moments', { content, images: images.value })
    moments.value.unshift(moment)
    draft.value = ''
    images.value = []
    composerOpen.value = false
    ui.toast('已发布')
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    posting.value = false
  }
}

async function toggleLike(moment: Moment) {
  if (busyLike.value === moment.id) return
  busyLike.value = moment.id
  try {
    const res = await api.post<{ liked: boolean; likes: User[] }>(
      `/moments/${moment.id}/like`,
    )
    moment.likedByMe = res.liked
    moment.likes = res.likes
  } catch (err) {
    ui.toast(errorText(err), 'error')
  } finally {
    busyLike.value = null
  }
}

function toggleComment(id: number) {
  commenting.value = commenting.value === id ? null : id
}

function onCommentKey(event: KeyboardEvent, moment: Moment) {
  if (event.key !== 'Enter' || event.isComposing) return
  event.preventDefault()
  void submitComment(moment)
}

async function submitComment(moment: Moment) {
  const content = (commentDrafts.value[moment.id] ?? '').trim()
  if (!content) return
  try {
    const comment = await api.post<MomentComment>(`/moments/${moment.id}/comments`, { content })
    moment.comments.push(comment)
    commentDrafts.value[moment.id] = ''
    commenting.value = null
  } catch (err) {
    ui.toast(errorText(err), 'error')
  }
}
</script>

<template>
  <div class="moments-page">
    <header class="page-head">
      <div class="head-left">
        <h2>朋友圈</h2>
        <span class="head-sub">{{ moments.length }} 条动态</span>
      </div>
      <button class="icon-btn" title="刷新" @click="load">
        <AppIcon name="refresh" :size="18" :class="{ spinning: loading }" />
      </button>
    </header>

    <div class="moments-scroll">
      <div class="moments-inner">
        <section class="composer-card">
          <AppAvatar :name="auth.user?.nickname ?? '我'" :src="auth.user?.avatar" :size="42" />
          <div class="composer-body">
            <textarea
              v-model="draft"
              :rows="composerOpen || draft ? 3 : 1"
              placeholder="这一刻的想法…"
              maxlength="2000"
              @focus="composerOpen = true"
            />
            <div v-if="images.length" class="preview-grid">
              <div v-for="(img, index) in images" :key="img" class="preview">
                <img :src="img" alt="待发布图片" />
                <button class="remove" title="移除" @click="images.splice(index, 1)">
                  <AppIcon name="close" :size="11" />
                </button>
              </div>
            </div>
            <div v-if="composerOpen || images.length" class="composer-actions">
              <button
                class="ghost-btn small"
                :disabled="uploading || images.length >= 9"
                @click="chooseImages"
              >
                <AppIcon name="image" :size="15" />
                {{ uploading ? '上传中…' : '图片' }}
              </button>
              <span class="muted count">{{ images.length }}/9</span>
              <button
                class="primary-btn small"
                :disabled="posting || uploading || (!draft.trim() && !images.length)"
                @click="publish"
              >
                {{ posting ? '发布中…' : '发表' }}
              </button>
            </div>
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              multiple
              hidden
              @change="onFiles"
            />
          </div>
        </section>

        <article v-for="moment in moments" :key="moment.id" class="moment-card">
          <AppAvatar :name="moment.user.nickname" :src="moment.user.avatar" :size="42" />
          <div class="moment-main">
            <div class="moment-name">{{ moment.user.nickname }}</div>
            <p v-if="moment.content" class="moment-text">{{ moment.content }}</p>
            <div
              v-if="moment.images.length"
              class="image-grid"
              :class="`n-${Math.min(moment.images.length, 4)}`"
            >
              <button
                v-for="img in moment.images"
                :key="img"
                class="moment-image"
                @click="preview = img"
              >
                <img :src="img" alt="朋友圈图片" />
              </button>
            </div>

            <div class="moment-meta">
              <span class="moment-time">{{ formatRelative(moment.createdAt) }}</span>
              <div class="moment-actions">
                <button
                  class="meta-btn"
                  :class="{ active: moment.likedByMe }"
                  @click="toggleLike(moment)"
                >
                  <AppIcon name="heart" :size="15" :filled="moment.likedByMe" />
                  {{ moment.likedByMe ? '已赞' : '赞' }}
                </button>
                <button class="meta-btn" @click="toggleComment(moment.id)">
                  <AppIcon name="comment" :size="15" />
                  评论
                </button>
              </div>
            </div>

            <div v-if="moment.likes.length || moment.comments.length" class="social-box">
              <div v-if="moment.likes.length" class="likes-row">
                <AppIcon name="heart" :size="12" filled />
                <span>{{ moment.likes.map((u) => u.nickname).join('、') }}</span>
              </div>
              <div v-if="moment.comments.length" class="comment-list">
                <p v-for="comment in moment.comments" :key="comment.id">
                  <b>{{ comment.user.nickname }}</b>：{{ comment.content }}
                </p>
              </div>
            </div>

            <div v-if="commenting === moment.id" class="comment-input">
              <input
                v-model="commentDrafts[moment.id]"
                placeholder="写下你的评论…"
                maxlength="500"
                @keydown="onCommentKey($event, moment)"
              />
              <button class="primary-btn small" @click="submitComment(moment)">发送</button>
            </div>
          </div>
        </article>

        <div v-if="!moments.length && !loading" class="empty-state moments-empty">
          <div class="empty-icon"><AppIcon name="discover" :size="28" /></div>
          <p>还没有动态，发布第一条吧</p>
        </div>
      </div>
    </div>

    <div v-if="preview" class="lightbox" @click="preview = ''">
      <img :src="preview" alt="图片预览" />
    </div>
  </div>
</template>

<style scoped>
.moments-page {
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
  justify-content: space-between;
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--line);
}

.head-left {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.head-left h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.head-sub {
  font-size: 11.5px;
  color: var(--muted);
}

.moments-scroll {
  flex: 1;
  overflow-y: auto;
}

.moments-inner {
  width: min(680px, 100%);
  margin: 0 auto;
  padding: 20px 22px 50px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.composer-card,
.moment-card {
  display: flex;
  gap: 13px;
  padding: 18px;
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 2px 10px rgba(20, 40, 30, 0.04);
}

.composer-body {
  flex: 1;
  min-width: 0;
}

.composer-body textarea {
  width: 100%;
  border: none;
  outline: none;
  resize: none;
  font-size: 14px;
  line-height: 1.6;
  background: transparent;
  padding: 8px 2px 4px;
  color: var(--text);
}

.composer-body textarea::placeholder {
  color: #b4b4b4;
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
  margin-top: 8px;
  max-width: 360px;
}

.preview {
  position: relative;
  aspect-ratio: 1;
  border-radius: 7px;
  overflow: hidden;
  background: #eee;
}

.preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.preview .remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.composer-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
}

.small {
  padding: 7px 13px;
  font-size: 12.5px;
  border-radius: 7px;
}

.count {
  font-size: 11px;
  margin-right: auto;
}

.moment-card {
  align-items: flex-start;
  animation: fade-in 0.22s ease both;
}

.moment-main {
  flex: 1;
  min-width: 0;
}

.moment-name {
  font-size: 14px;
  font-weight: 600;
  color: #3a5a8c;
}

.moment-text {
  margin: 5px 0 0;
  font-size: 14px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.image-grid {
  display: grid;
  gap: 5px;
  margin-top: 10px;
  max-width: 360px;
}

.image-grid.n-1 {
  grid-template-columns: minmax(0, 240px);
}

.image-grid.n-2,
.image-grid.n-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.image-grid.n-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.image-grid.n-4 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.moment-image {
  padding: 0;
  border-radius: 5px;
  overflow: hidden;
  background: #eee;
  aspect-ratio: 1;
}

.image-grid.n-1 .moment-image {
  aspect-ratio: 4 / 3;
}

.moment-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.2s ease;
}

.moment-image:hover img {
  transform: scale(1.03);
}

.moment-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 9px;
}

.moment-time {
  font-size: 11.5px;
  color: #b0b0b0;
}

.moment-actions {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.meta-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 11.5px;
  color: #8a8a8a;
}

.meta-btn:hover {
  background: #f4f4f4;
  color: #4a4a4a;
}

.meta-btn.active {
  color: var(--danger);
}

.social-box {
  margin-top: 9px;
  padding: 8px 11px;
  border-radius: 7px;
  background: #f6f7f6;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.likes-row {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #5c7fa8;
  font-size: 12px;
  line-height: 1.5;
}

.likes-row svg {
  color: var(--danger);
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
  border-top: 1px solid #ececec;
  padding-top: 5px;
}

.likes-row + .comment-list {
  margin-top: 2px;
}

.comment-list p {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.6;
  color: #333;
  word-break: break-word;
}

.comment-list b {
  color: #3a5a8c;
  font-weight: 500;
}

.comment-input {
  display: flex;
  gap: 8px;
  margin-top: 9px;
}

.comment-input input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: #f4f4f4;
  border-radius: 7px;
  padding: 8px 11px;
  font-size: 12.5px;
}

.moments-empty {
  padding: 60px 0;
  height: auto;
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
}

@media (max-width: 860px) {
  .page-head {
    padding: 0 16px;
    height: 56px;
  }

  .moments-inner {
    padding: 14px 12px 40px;
    gap: 10px;
  }

  .composer-card,
  .moment-card {
    padding: 14px;
    gap: 10px;
    border-radius: 12px;
  }

  .image-grid {
    max-width: 100%;
  }

  .image-grid.n-1 {
    grid-template-columns: minmax(0, 200px);
  }
}
</style>
