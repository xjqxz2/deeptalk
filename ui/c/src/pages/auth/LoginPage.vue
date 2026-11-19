<script setup lang="ts">
/** 登录页：一屏完成身份确认，错误统一提示，不泄露账号是否存在。 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  CheckCircleFilled,
  InfoCircleOutlined,
  LockOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons-vue'
import { ApiError } from '@deeptalk/shared'
import AuthLayout from '@/layouts/AuthLayout.vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

/** 首次部署默认填入管理员邮箱，密码仍需用户输入。 */
const email = ref('xjqxz_2@163.com')
const password = ref('')
const error = ref('')
const loading = ref(false)
const forgotOpen = ref(false)
const legalOpen = ref(false)
const passwordRef = ref<{ focus: () => void } | null>(null)

const canSubmit = computed(() => email.value.trim().length > 0 && password.value.length > 0 && !loading.value)

/** 登录：错误统一展示在表单内，成功回跳来源页。 */
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
    await userStore.login(email.value.trim(), password.value)
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.push(redirect)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '登录失败，请稍后重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout
    eyebrow="ACCOUNT ACCESS"
    title="欢迎回来"
    subtitle="登录后继续你的 DeepTalk 工作台。"
    switch-text="还没有账号？"
    switch-label="立即注册"
    switch-to="/register"
  >
    <form class="auth-form" novalidate @submit.prevent="onSubmit">
      <div class="field">
        <label class="field-label" for="login-email">邮箱</label>
        <a-input
          id="login-email"
          v-model:value="email"
          class="auth-input"
          size="large"
          placeholder="you@company.com"
          autocomplete="username"
          @press-enter="passwordRef?.focus()"
        >
          <template #prefix><MailOutlined class="field-icon" /></template>
        </a-input>
      </div>

      <div class="field">
        <div class="field-head">
          <label class="field-label" for="login-password">密码</label>
          <button type="button" class="text-link" @click="forgotOpen = true">忘记密码？</button>
        </div>
        <a-input-password
          id="login-password"
          ref="passwordRef"
          v-model:value="password"
          class="auth-input"
          size="large"
          placeholder="请输入密码"
          autocomplete="current-password"
          @press-enter="onSubmit"
        >
          <template #prefix><LockOutlined class="field-icon" /></template>
        </a-input-password>
      </div>

      <div v-if="error" class="form-error" role="alert" aria-live="polite">
        {{ error }}
      </div>

      <a-button type="primary" html-type="submit" block class="submit-btn" :loading="loading" :disabled="!canSubmit">
        登录
      </a-button>

      <p class="signin-hint">
        <CheckCircleFilled />
        登录后进入你的对话空间
      </p>

      <p class="terms">
        登录即表示同意
        <button type="button" class="inline-link" @click="legalOpen = true">《服务条款》</button>
        与
        <button type="button" class="inline-link" @click="legalOpen = true">《隐私政策》</button>
      </p>

      <p class="note">
        <InfoCircleOutlined />
        无需验证邮件 · 忘记密码由管理员重置
      </p>

      <div class="security">
        <SafetyCertificateOutlined />
        全程 HTTPS · 会话 HttpOnly，默认 7 天
      </div>
    </form>

    <a-modal v-model:open="forgotOpen" title="忘记密码" :footer="null" :width="420">
      <p class="modal-text">
        本项目不发送验证邮件，也不支持邮件找回密码。请联系平台管理员在「用户管理」中重置密码，或在服务器上使用 CLI 强制重置。
      </p>
      <a-button type="primary" block @click="forgotOpen = false">我知道了</a-button>
    </a-modal>

    <a-modal v-model:open="legalOpen" title="服务条款与隐私政策" :footer="null" :width="480">
      <div class="legal">
        <p>DeepTalk 是个人 AI 对话工作台。你上传或输入的内容仅用于生成本次回答，并按账号隔离存储。</p>
        <p>管理后台只展示账号、用量与最近登录信息，不读取对话正文；导出的 CSV 仅包含账号字段。</p>
        <p>当前为 MVP 版本，服务条款与隐私政策为演示文案，正式上线前需由法务补充。</p>
      </div>
      <a-button type="primary" block @click="legalOpen = false">关闭</a-button>
    </a-modal>
  </AuthLayout>
</template>

<style scoped>
.auth-form {
  display: flex;
  flex-direction: column;
}

.field {
  margin-bottom: 18px;
}

.field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.field-label {
  display: block;
  margin-bottom: 8px;
  color: var(--dt-text-secondary);
  font-size: 12px;
  font-weight: 620;
}

.field-icon {
  color: var(--dt-text-faint);
  font-size: 13px;
}

.auth-input :deep(.ant-input-affix-wrapper),
.auth-input :deep(.ant-input) {
  min-height: 46px;
  border-radius: 10px;
  font-size: 13.2px;
}

.auth-input :deep(.ant-input-affix-wrapper) {
  padding-inline: 13px;
}

.auth-input :deep(.ant-input-prefix) {
  margin-inline-end: 9px;
}

.text-link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-brand);
  font-size: 10.4px;
  cursor: pointer;
}

.text-link:hover {
  color: var(--dt-brand-hover);
}

.form-error {
  margin: -4px 0 12px;
  padding: 9px 12px;
  border: 1px solid #fee4e2;
  border-radius: 9px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11.8px;
  line-height: 18px;
}

.submit-btn {
  height: 46px;
  margin-top: 2px;
  border-radius: 10px;
  font-size: 13.6px;
  font-weight: 620;
}

.signin-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 14px 0 0;
  color: var(--dt-text-muted);
  font-size: 10.8px;
}

.signin-hint :deep(svg) {
  color: var(--dt-success);
  font-size: 10px;
}

.terms {
  margin: 18px 0 0;
  color: var(--dt-text-faint);
  font-size: 10.2px;
  line-height: 18px;
}

.inline-link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--dt-text-muted);
  font-size: 10.2px;
  text-decoration: underline;
  text-underline-offset: 2px;
  cursor: pointer;
}

.note {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 20px 0 0;
  padding-top: 16px;
  border-top: 1px solid var(--dt-line-soft);
  color: var(--dt-text-muted);
  font-size: 10.6px;
}

.note :deep(svg) {
  color: var(--dt-brand);
  font-size: 10px;
}

.security {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 14px;
  color: var(--dt-text-faint);
  font-size: 10.2px;
}

.modal-text {
  margin: 0 0 18px;
  color: var(--dt-text-secondary);
  font-size: 13px;
  line-height: 22px;
}

.legal {
  color: var(--dt-text-secondary);
  font-size: 12.6px;
  line-height: 21px;
  margin-bottom: 16px;
}

.legal p {
  margin: 0 0 10px;
}

@media (max-width: 480px) {
  .auth-input :deep(.ant-input-affix-wrapper),
  .auth-input :deep(.ant-input) {
    min-height: 44px;
  }

  .submit-btn {
    height: 44px;
  }
}
</style>
