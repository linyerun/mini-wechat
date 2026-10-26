<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { errorText } from '@/utils/format'

const router = useRouter()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const loading = ref(false)
const error = ref('')
const form = reactive({ username: 'alice', password: '123456', nickname: '' })

const demos = [
  { username: 'alice', name: '林小敏' },
  { username: 'bob', name: '陈杰' },
  { username: 'carol', name: '苏晴' },
]

async function submit(username = form.username, password = form.password) {
  error.value = ''
  if (!username.trim() || !password) {
    error.value = '请输入用户名和密码'
    return
  }
  if (mode.value === 'register' && !form.nickname.trim()) {
    error.value = '请输入昵称'
    return
  }
  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(username.trim(), password)
    } else {
      await auth.register(username.trim(), password, form.nickname.trim())
    }
    await router.replace('/')
  } catch (err) {
    error.value = errorText(err)
  } finally {
    loading.value = false
  }
}

function switchMode(next: 'login' | 'register') {
  if (mode.value === next) return
  mode.value = next
  error.value = ''
  if (next === 'register') {
    form.username = ''
    form.password = ''
  }
}

async function quickLogin(demo: { username: string }) {
  mode.value = 'login'
  form.username = demo.username
  form.password = '123456'
  await submit(demo.username, '123456')
}
</script>

<template>
  <div class="login-page">
    <div class="login-shell">
      <section class="brand-pane">
        <div class="brand-top">
          <div class="brand-logo">
            <svg viewBox="0 0 48 48" width="34" height="34" aria-hidden="true">
              <path
                d="M20 8C11 8 4 13.6 4 20.5c0 3.9 2.3 7.4 6 9.7L8.6 35l5.2-2.6c1.9.6 3.9.9 6.2.9 1.4 0 2.8-.1 4-.4"
                fill="#fff"
              />
              <path
                d="M44 30.5c0-5.5-5.4-10-12-10s-12 4.5-12 10 5.4 10 12 10c1.4 0 2.8-.2 4.1-.5l4.3 2.1-1.2-4c2.9-1.9 4.8-4.6 4.8-7.6z"
                fill="#fff"
              />
              <circle cx="15" cy="18.5" r="1.8" fill="#07c160" />
              <circle cx="25" cy="18.5" r="1.8" fill="#07c160" />
              <circle cx="28" cy="30" r="1.5" fill="#07c160" />
              <circle cx="36" cy="30" r="1.5" fill="#07c160" />
            </svg>
          </div>
          <div class="brand-title">
            <h1>微信 Mini</h1>
            <p>Go · Vue 3 · TypeScript</p>
          </div>
        </div>

        <div class="brand-hero">
          <div class="phone-mock">
            <div class="phone-head">
              <span class="phone-dot" />
              周末爬山小队
              <span class="phone-count">4 人</span>
            </div>
            <div class="chat-line">这周六早上 8 点老地方集合～</div>
            <div class="chat-line self">收到，我带咖啡 ☕</div>
            <div class="chat-line">山顶的风也太舒服了</div>
            <div class="phone-input">说点什么…</div>
          </div>
        </div>

        <div class="brand-foot">
          <span>实时消息</span>
          <i />
          <span>群聊</span>
          <i />
          <span>朋友圈</span>
        </div>
      </section>

      <section class="form-pane">
        <div class="form-inner">
          <div class="mobile-brand">
            <div class="mini-logo">
              <svg viewBox="0 0 48 48" width="24" height="24" aria-hidden="true">
                <path
                  d="M20 8C11 8 4 13.6 4 20.5c0 3.9 2.3 7.4 6 9.7L8.6 35l5.2-2.6c1.9.6 3.9.9 6.2.9 1.4 0 2.8-.1 4-.4"
                  fill="#fff"
                />
                <path
                  d="M44 30.5c0-5.5-5.4-10-12-10s-12 4.5-12 10 5.4 10 12 10c1.4 0 2.8-.2 4.1-.5l4.3 2.1-1.2-4c2.9-1.9 4.8-4.6 4.8-7.6z"
                  fill="#fff"
                />
              </svg>
            </div>
            <span>微信 Mini</span>
          </div>

          <h2>{{ mode === 'login' ? '欢迎回来' : '创建账号' }}</h2>
          <p class="form-sub">
            {{ mode === 'login' ? '登录后，和重要的人保持联系' : '注册一个账号，加入 mini 微信' }}
          </p>

          <div class="mode-tabs">
            <button :class="{ active: mode === 'login' }" @click="switchMode('login')">登录</button>
            <button :class="{ active: mode === 'register' }" @click="switchMode('register')">
              注册
            </button>
          </div>

          <form @submit.prevent="submit()">
            <label class="field">
              <span>用户名</span>
              <input
                v-model="form.username"
                autocomplete="username"
                placeholder="请输入用户名"
                maxlength="20"
              />
            </label>
            <label v-if="mode === 'register'" class="field">
              <span>昵称</span>
              <input v-model="form.nickname" placeholder="别人看到的名字" maxlength="20" />
            </label>
            <label class="field">
              <span>密码</span>
              <input
                v-model="form.password"
                type="password"
                :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
                placeholder="请输入密码"
                maxlength="32"
              />
            </label>

            <p v-if="error" class="form-error">{{ error }}</p>

            <button class="primary-btn submit-btn" type="submit" :disabled="loading">
              {{ loading ? '请稍候…' : mode === 'login' ? '登 录' : '注册并进入' }}
            </button>
          </form>

          <div class="demo-block">
            <p>体验账号 · 密码均为 123456</p>
            <div class="demo-list">
              <button
                v-for="demo in demos"
                :key="demo.username"
                class="demo-chip"
                :disabled="loading"
                @click="quickLogin(demo)"
              >
                <span class="chip-dot" />
                {{ demo.name }}
              </button>
            </div>
          </div>

          <p class="legal">本地演示项目 · 数据保存在本机 SQLite</p>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 22px;
  background:
    radial-gradient(900px 500px at 8% -10%, rgba(7, 193, 96, 0.16), transparent 60%),
    radial-gradient(700px 500px at 100% 110%, rgba(16, 174, 255, 0.14), transparent 60%),
    #f3f6f3;
}

.login-shell {
  width: min(960px, 100%);
  min-height: min(640px, calc(100dvh - 44px));
  display: grid;
  grid-template-columns: 1.04fr 1fr;
  border-radius: 26px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 30px 80px rgba(15, 45, 30, 0.16);
}

.brand-pane {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 40px 42px 34px;
  color: #fff;
  background:
    radial-gradient(420px 300px at 90% 0%, rgba(255, 255, 255, 0.16), transparent 60%),
    radial-gradient(380px 300px at -10% 100%, rgba(0, 0, 0, 0.16), transparent 60%),
    linear-gradient(155deg, #0abb5e, #07c160 46%, #069c53);
  overflow: hidden;
}

.brand-pane::after {
  content: '';
  position: absolute;
  width: 320px;
  height: 320px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.18);
  right: -120px;
  bottom: -140px;
}

.brand-top {
  display: flex;
  align-items: center;
  gap: 14px;
  position: relative;
  z-index: 1;
}

.brand-logo {
  width: 54px;
  height: 54px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.18);
  display: grid;
  place-items: center;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.25);
}

.brand-title h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.brand-title p {
  margin: 3px 0 0;
  font-size: 11.5px;
  letter-spacing: 1.2px;
  opacity: 0.78;
  text-transform: uppercase;
}

.brand-hero {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 26px 0;
  position: relative;
  z-index: 1;
}

.phone-mock {
  width: min(300px, 100%);
  padding: 16px 15px 14px;
  border-radius: 22px;
  background: rgba(8, 60, 34, 0.3);
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.24),
    0 24px 50px rgba(0, 40, 20, 0.24);
  backdrop-filter: blur(6px);
}

.phone-head {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  opacity: 0.94;
  padding: 2px 4px 14px;
}

.phone-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #b7ffd8;
}

.phone-count {
  margin-left: auto;
  font-size: 10px;
  opacity: 0.7;
}

.chat-line {
  width: fit-content;
  max-width: 84%;
  margin-bottom: 9px;
  padding: 8px 12px;
  border-radius: 12px 12px 12px 4px;
  background: rgba(255, 255, 255, 0.94);
  color: #16301f;
  font-size: 12.5px;
  line-height: 1.5;
  animation: fade-in 0.5s ease both;
}

.chat-line.self {
  margin-left: auto;
  background: var(--bubble-self);
  border-radius: 12px 12px 4px 12px;
  animation-delay: 0.08s;
}

.chat-line:nth-child(4) {
  animation-delay: 0.16s;
}

.phone-input {
  margin-top: 12px;
  padding: 7px 10px;
  border-radius: 9px;
  border: 1px solid rgba(255, 255, 255, 0.28);
  color: rgba(255, 255, 255, 0.65);
  font-size: 11.5px;
}

.brand-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 11px;
  letter-spacing: 1.2px;
  opacity: 0.72;
  position: relative;
  z-index: 1;
}

.brand-foot i {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
}

.form-pane {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 46px;
  background: #fff;
}

.form-inner {
  width: min(330px, 100%);
}

.mobile-brand {
  display: none;
  align-items: center;
  gap: 9px;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 26px;
}

.mini-logo {
  width: 38px;
  height: 38px;
  border-radius: 11px;
  background: var(--green);
  display: grid;
  place-items: center;
}

.form-inner h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: 0.4px;
}

.form-sub {
  margin: 7px 0 0;
  font-size: 13px;
  color: var(--muted);
}

.mode-tabs {
  display: flex;
  gap: 4px;
  background: #f3f3f3;
  border-radius: 10px;
  padding: 4px;
  margin: 22px 0 14px;
}

.mode-tabs button {
  flex: 1;
  padding: 8px 0;
  font-size: 13px;
  color: var(--text-2);
  border-radius: 7px;
  transition: all 0.16s ease;
}

.mode-tabs button.active {
  background: #fff;
  color: var(--text);
  font-weight: 500;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.09);
}

.form-inner .field {
  margin-top: 12px;
}

.form-inner .field span {
  font-size: 12px;
  color: var(--text-2);
}

.form-inner .field input {
  padding: 11px 12px;
  border-radius: 9px;
}

.form-error {
  margin: 10px 0 0;
  font-size: 12.5px;
  color: var(--danger);
}

.submit-btn {
  width: 100%;
  height: 44px;
  margin-top: 18px;
  font-size: 15px;
  letter-spacing: 2px;
  border-radius: 9px;
}

.demo-block {
  margin-top: 24px;
  padding-top: 18px;
  border-top: 1px dashed var(--line);
}

.demo-block > p {
  margin: 0 0 10px;
  font-size: 11.5px;
  color: var(--muted);
}

.demo-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.demo-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 11px;
  border-radius: 20px;
  background: #f5f6f5;
  border: 1px solid #ededed;
  font-size: 12px;
  color: var(--text-2);
  transition: all 0.15s ease;
}

.demo-chip:hover:not(:disabled) {
  border-color: var(--green);
  color: var(--green);
  background: var(--green-soft);
}

.demo-chip:disabled {
  opacity: 0.5;
  cursor: default;
}

.chip-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--green);
}

.legal {
  margin: 22px 0 0;
  font-size: 10.5px;
  color: #b9b9b9;
  text-align: center;
}

@media (max-width: 900px) {
  .login-page {
    padding: 0;
    align-items: stretch;
  }

  .login-shell {
    grid-template-columns: 1fr;
    border-radius: 0;
    min-height: 100dvh;
    box-shadow: none;
  }

  .brand-pane {
    display: none;
  }

  .form-pane {
    padding: 46px 26px 36px;
    align-items: flex-start;
  }

  .form-inner {
    width: min(360px, 100%);
    margin: 0 auto;
  }

  .mobile-brand {
    display: flex;
  }
}
</style>
