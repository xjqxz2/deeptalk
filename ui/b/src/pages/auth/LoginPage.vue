<script setup lang="ts">
/** 管理员登录：仅平台管理员可进入控制台。 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined, LockOutlined, MailOutlined, SafetyCertificateOutlined, WarningOutlined } from '@ant-design/icons-vue'
import { ApiError, BrandMark } from '@deeptalk/shared'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

/** 被守卫拦截的非管理员提示。 */
const forbidden = computed(() => route.query.forbidden === '1' && !userStore.isAdmin)
const canSubmit = computed(() => email.value.trim().length > 0 && password.value.length > 0 && !loading.value)

/** 登录并校验管理员角色。 */
async function onSubmit(): Promise<void> {
  error.value = ''
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) {
    error.value = '请输入正确的邮箱地址'
    return
  }
  if (!password.value) {
    error.value = '请输入密码'
    return
  }
  loading.value = true
  try {
    const user = await userStore.login(email.value.trim(), password.value)
    if (user.role !== 'admin') {
      await userStore.logout()
      error.value = '该账号不是平台管理员，无法进入控制台'
      return
    }
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/overview'
    await router.push(redirect)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '登录失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

/** 退出当前非管理员账号。 */
async function signOutCurrent(): Promise<void> {
  await userStore.logout()
  await router.replace({ name: 'login' })
}
</script>

<template>
  <div class="admin-login">
    <div class="grid-bg" aria-hidden="true" />
    <div class="login-card">
      <div class="brand">
        <BrandMark :size="34" :radius="9.5" />
        <span class="brand-name">DeepTalk</span>
        <span class="admin-badge">ADMIN</span>
      </div>

      <p class="eyebrow">PLATFORM CONSOLE</p>
      <h1>登录平台控制台</h1>
      <p class="subtitle">使用平台管理员账号登录，管理厂商、模型与用户。</p>

      <div v-if="forbidden" class="forbidden" role="alert">
        <WarningOutlined />
        <div>
          <strong>当前账号不是平台管理员</strong>
          <p>首个注册账号为管理员，其余账号只能使用对话工作台。</p>
          <button type="button" class="text-btn" @click="signOutCurrent">退出当前账号</button>
        </div>
      </div>

      <form novalidate @submit.prevent="onSubmit">
        <div class="field">
          <label for="admin-email">邮箱</label>
          <a-input
            id="admin-email"
            v-model:value="email"
            size="large"
            class="login-input"
            placeholder="admin@company.com"
            autocomplete="username"
          >
            <template #prefix><MailOutlined class="input-icon" /></template>
          </a-input>
        </div>
        <div class="field">
          <label for="admin-password">密码</label>
          <a-input-password
            id="admin-password"
            v-model:value="password"
            size="large"
            class="login-input"
            placeholder="请输入密码"
            autocomplete="current-password"
            @press-enter="onSubmit"
          >
            <template #prefix><LockOutlined class="input-icon" /></template>
          </a-input-password>
        </div>

        <div v-if="error" class="form-error" role="alert">{{ error }}</div>

        <a-button type="primary" html-type="submit" block class="submit" :loading="loading" :disabled="!canSubmit">
          登录控制台
        </a-button>
      </form>

      <p class="secure">
        <SafetyCertificateOutlined />
        仅平台管理员可访问 · 会话 HttpOnly，默认 7 天
      </p>

      <a class="workspace-link" href="/">
        <ArrowLeftOutlined />
        返回对话工作台
      </a>
    </div>

    <p class="copyright">© 2026 DeepTalk · Admin Console v1.0.0</p>
  </div>
</template>

<style scoped>
.admin-login {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 32px;
  overflow: hidden;
  background:
    radial-gradient(circle at 18% 12%, #e8eeff 0%, rgba(232, 238, 255, 0) 46%),
    radial-gradient(circle at 82% 88%, #eef3ff 0%, rgba(238, 243, 255, 0) 42%),
    #f7f8fa;
}

.grid-bg {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(#d6dce6 1px, transparent 1px);
  background-size: 26px 26px;
  opacity: 0.45;
  mask-image: radial-gradient(circle at 50% 45%, #000 0%, transparent 72%);
}

.login-card {
  position: relative;
  z-index: 1;
  width: 420px;
  max-width: 100%;
  padding: 30px 30px 24px;
  border: 1px solid #e7eaf1;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 20px 44px rgba(16, 24, 40, 0.1);
  backdrop-filter: blur(8px);
}

.brand {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 26px;
}

.brand-name {
  color: var(--dt-text);
  font-size: 15.4px;
  font-weight: 730;
  letter-spacing: -0.2px;
}

.admin-badge {
  padding: 2px 6px;
  border-radius: 5px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 8.6px;
  font-weight: 700;
  letter-spacing: 0.6px;
}

.eyebrow {
  margin: 0 0 8px;
  color: var(--dt-text-faint);
  font-size: 9.6px;
  font-weight: 650;
  letter-spacing: 1.5px;
}

h1 {
  margin: 0 0 8px;
  color: var(--dt-text);
  font-size: 26px;
  line-height: 1.3;
  font-weight: 740;
  letter-spacing: -0.5px;
}

.subtitle {
  margin: 0 0 24px;
  color: var(--dt-text-muted);
  font-size: 12.4px;
  line-height: 19px;
}

.field {
  margin-bottom: 16px;
}

.field label {
  display: block;
  margin-bottom: 7px;
  color: var(--dt-text-secondary);
  font-size: 11.6px;
  font-weight: 620;
}

.login-input :deep(.ant-input-affix-wrapper),
.login-input :deep(.ant-input) {
  min-height: 44px;
  border-radius: 10px;
  font-size: 13px;
}

.input-icon {
  color: var(--dt-text-faint);
  font-size: 12.6px;
}

.form-error {
  margin: -2px 0 12px;
  padding: 8px 11px;
  border: 1px solid #fee4e2;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11.4px;
  line-height: 17px;
}

.submit {
  height: 44px;
  margin-top: 2px;
  border-radius: 10px;
  font-size: 13.2px;
  font-weight: 620;
}

.forbidden {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-bottom: 18px;
  padding: 11px 12px;
  border: 1px solid #fee4e2;
  border-radius: 10px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.forbidden :deep(svg) {
  margin-top: 2px;
}

.forbidden strong {
  font-size: 11.8px;
}

.forbidden p {
  margin: 3px 0 4px;
  font-size: 10.8px;
  line-height: 16px;
}

.text-btn {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-danger-text);
  font-size: 10.8px;
  text-decoration: underline;
  cursor: pointer;
}

.secure {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 20px 0 16px;
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.secure :deep(svg) {
  color: var(--dt-success);
}

.workspace-link {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding-top: 14px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-muted);
  font-size: 10.8px;
}

.workspace-link:hover {
  color: var(--dt-brand-hover);
}

.copyright {
  position: relative;
  z-index: 1;
  margin: 22px 0 0;
  color: var(--dt-text-faint);
  font-size: 9.6px;
}

@media (max-width: 480px) {
  .admin-login {
    padding: 18px;
  }

  .login-card {
    padding: 24px 20px 20px;
    border-radius: 16px;
  }

  h1 {
    font-size: 23px;
  }
}
</style>
