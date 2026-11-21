<script setup lang="ts">
/** 重置密码弹窗：生成一次性密码并只展示一次。 */
import { computed, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { CopyOutlined, KeyOutlined, WarningOutlined } from '@ant-design/icons-vue'
import { ApiError, copyText } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'
import { resetUserPassword } from '@/services/admin'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ user: User | null }>()
const emit = defineEmits<{ (event: 'done'): void }>()

const loading = ref(false)
const password = ref('')
const error = ref('')
const copied = ref(false)

const canReset = computed(() => Boolean(props.user))

watch(open, (value) => {
  if (!value) {
    password.value = ''
    error.value = ''
    copied.value = false
  }
})

/** 生成一次性密码。 */
async function reset(): Promise<void> {
  if (!props.user) return
  loading.value = true
  error.value = ''
  try {
    const data = await resetUserPassword(props.user.id)
    password.value = data.password
    emit('done')
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '重置失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

/** 复制一次性密码。 */
async function copyPassword(): Promise<void> {
  const ok = await copyText(password.value)
  if (ok) {
    copied.value = true
    message.success('已复制')
    window.setTimeout(() => {
      copied.value = false
    }, 1600)
  }
}
</script>

<template>
  <a-modal v-model:open="open" title="重置密码" :width="440" :footer="null" destroy-on-close>
    <div class="reset">
      <div class="user-line">
        <span class="avatar">{{ user?.avatar_text ?? 'U' }}</span>
        <div>
          <strong>{{ user?.nickname }}</strong>
          <span>{{ user?.email }}</span>
        </div>
      </div>

      <div v-if="!password" class="warn">
        <WarningOutlined />
        <p>
          将为该账号生成一次性强密码，并立即撤销全部已登录会话。旧密码随即失效，系统不会发送邮件。
        </p>
      </div>

      <div v-if="password" class="password-box">
        <div class="password-head">
          <KeyOutlined />
          <span>一次性密码（只显示一次）</span>
        </div>
        <div class="password-value">
          <code>{{ password }}</code>
          <a-button size="small" @click="copyPassword">
            <CopyOutlined />
            {{ copied ? '已复制' : '复制' }}
          </a-button>
        </div>
        <p class="password-hint">请立即转交用户。关闭弹窗后无法再次查看，只能重新生成。</p>
      </div>

      <div v-if="error" class="form-error" role="alert">{{ error }}</div>

      <div class="actions">
        <a-button @click="open = false">{{ password ? '完成' : '取消' }}</a-button>
        <a-button v-if="!password" type="primary" danger :loading="loading" :disabled="!canReset" @click="reset">
          生成并重置
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.reset {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.user-line {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--dt-sidebar);
  border: 1px solid var(--dt-line-soft);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 12.4px;
  font-weight: 700;
}

.user-line div {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.user-line strong {
  color: var(--dt-text);
  font-size: 12.4px;
}

.user-line span {
  color: var(--dt-text-faint);
  font-size: 10.4px;
}

.warn {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--dt-danger-soft);
  border-radius: 9px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
}

.warn :deep(svg) {
  margin-top: 2px;
}

.warn p {
  margin: 0;
  font-size: 11px;
  line-height: 18px;
}

.password-box {
  padding: 12px;
  border: 1px solid var(--dt-success);
  border-radius: 10px;
  background: var(--dt-success-soft);
}

.password-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: var(--dt-success-text);
  font-size: 10.6px;
  font-weight: 600;
}

.password-value {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 11px;
  border: 1px dashed #a6f4c5;
  border-radius: 8px;
  background: var(--dt-surface);
}

.password-value code {
  color: var(--dt-text);
  font-family: var(--dt-mono);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.4px;
}

.password-hint {
  margin: 8px 0 0;
  color: var(--dt-success-text);
  font-size: 9.8px;
  line-height: 15px;
}

.form-error {
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 4px;
}
</style>
