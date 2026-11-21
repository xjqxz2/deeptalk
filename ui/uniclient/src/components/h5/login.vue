<template>
  <div class="auth-shell auth-mobile">
    <section class="auth-panel">
      <div class="panel-top">
        <div class="brand-inline">
          <img class="brand-logo-img" src="/static/shark.png" alt="DeepTalk" />
          <span class="brand-name">DeepTalk</span>
        </div>
        <div class="switch">
          <span>{{ mode === 'login' ? '还没有账号？' : '已有账号？' }}</span>
          <Btn class="switch-link" @click="toggleMode">{{ mode === 'login' ? '立即注册' : '去登录' }}</Btn>
        </div>
      </div>

      <div class="form-wrap">
        <div class="form-inner">
          <p class="eyebrow">ACCOUNT ACCESS</p>
          <h1>{{ mode === 'login' ? '欢迎回来' : '创建账号' }}</h1>
          <p class="subtitle">{{ mode === 'login' ? '登录后继续你的 DeepTalk 工作台。' : '注册后自动登录，立即开始对话。' }}</p>

          <form class="auth-form" novalidate @submit.prevent="submit">
            <div class="field">
              <label class="field-label" for="login-email">邮箱</label>
              <NativeInput
                class="native-auth-input"
                :model-value="email"
                type="email"
                placeholder="you@company.com"
                aria-label="邮箱"
                @update:model-value="(value) => (email = value)"
              />
            </div>

            <div class="field">
              <div class="field-head">
                <label class="field-label" for="login-password">密码</label>
                <span class="field-link">忘记密码？</span>
              </div>
              <NativeInput
                class="native-auth-input"
                :model-value="password"
                type="password"
                :placeholder="mode === 'login' ? '请输入密码' : '至少 8 位，建议含字母与数字'"
                aria-label="密码"
                @update:model-value="(value) => (password = value)"
              />
            </div>

            <p v-if="error" class="form-error">{{ error }}</p>

            <Btn class="submit-btn primary-btn" type="submit" :disabled="loading" @click="submit">
              {{ loading ? '请稍候…' : mode === 'login' ? '登录' : '注册并登录' }}
            </Btn>
          </form>

          <p class="auth-note">登录即表示同意《服务条款》与《隐私政策》</p>
          <ul class="auth-hints">
            <li>无需验证邮件，忘记密码可由管理员重置</li>
            <li>全程 HTTPS，会话保存在本地，可随时退出</li>
          </ul>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
/** H5 登录页：套用 PWA AuthLayout 的面板结构与样式，手机上隐藏左侧品牌区。 */
import { computed, ref } from 'vue'
import { login, register } from '@/api/auth'
import { setSession } from '@/store/session'
import { goChat } from '@/utils/nav'
import Btn from './btn.vue'
import NativeInput from './native-input.vue'

const mode = ref('login')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const canSubmit = computed(() => email.value.trim().length > 0 && password.value.length > 0)

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  error.value = ''
}

async function submit() {
  if (!canSubmit.value || loading.value) return
  error.value = ''
  loading.value = true
  try {
    const data = mode.value === 'login'
      ? await login(email.value.trim(), password.value)
      : await register(email.value.trim(), password.value)
    setSession(data.token, data.user)
    goChat()
  } catch (err) {
    error.value = err?.message || '操作失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>
