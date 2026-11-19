<script setup lang="ts">
/** 账号设置弹窗：展示账号信息、修改密码与退出登录。 */
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { LockOutlined, LogoutOutlined } from '@ant-design/icons-vue'
import { ApiError, ROLE_LABELS, planLabel } from '@deeptalk/shared'
import { changePassword } from '@/services/auth'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'

/** 弹窗开关（v-model:open）。 */
const open = defineModel<boolean>('open', { default: false })

const userStore = useUserStore()
const chatStore = useChatStore()
const router = useRouter()

const form = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const submitting = ref(false)
const error = ref('')

const user = computed(() => userStore.user)

/** 关闭弹窗并重置表单。 */
function close(): void {
  open.value = false
}

watch(open, (value) => {
  if (!value) {
    form.oldPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    error.value = ''
  }
})

/** 提交改密；成功后服务端撤销全部会话，需要重新登录。 */
async function submit(): Promise<void> {
  error.value = ''
  if (!form.oldPassword) {
    error.value = '请输入当前密码'
    return
  }
  if (form.newPassword.length < 8 || !/[A-Za-z]/.test(form.newPassword) || !/\d/.test(form.newPassword)) {
    error.value = '新密码至少 8 位，且需同时包含字母与数字'
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }
  submitting.value = true
  try {
    await changePassword(form.oldPassword, form.newPassword)
    message.success('密码已更新，请重新登录')
    chatStore.reset()
    // changePassword 已撤销会话；这里只清理前端状态，不再调用退出接口
    userStore.$patch({ user: null, usage: null })
    close()
    await router.push({ name: 'login' })
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '修改失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

/** 退出登录。 */
async function logout(): Promise<void> {
  await userStore.logout()
  chatStore.reset()
  close()
  await router.push({ name: 'login' })
}
</script>

<template>
  <a-modal
    :open="open"
    title="账号设置"
    :footer="null"
    :width="468"
    destroy-on-close
    @cancel="close"
  >
    <div v-if="user" class="account">
      <div class="account-head">
        <span class="avatar">{{ user.avatar_text }}</span>
        <div class="identity">
          <strong>{{ user.nickname }}</strong>
          <span>{{ user.email }}</span>
        </div>
        <span class="role-tag">{{ ROLE_LABELS[user.role] ?? user.role }}</span>
      </div>

      <div class="info-grid">
        <div class="info-item">
          <span class="info-label">套餐</span>
          <span class="info-value">{{ planLabel(user.plan) }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">注册时间</span>
          <span class="info-value">{{ user.created_at.slice(0, 10) }}</span>
        </div>
      </div>

      <div class="section-title">
        <LockOutlined />
        修改密码
      </div>
      <p class="section-hint">修改成功后，所有设备上的会话都会立即失效。</p>
      <a-form layout="vertical" @submit.prevent="submit">
        <a-form-item label="当前密码">
          <a-input-password v-model:value="form.oldPassword" placeholder="请输入当前密码" autocomplete="current-password" />
        </a-form-item>
        <a-form-item label="新密码">
          <a-input-password v-model:value="form.newPassword" placeholder="至少 8 位，包含字母与数字" autocomplete="new-password" />
        </a-form-item>
        <a-form-item label="确认新密码">
          <a-input-password v-model:value="form.confirmPassword" placeholder="再次输入新密码" autocomplete="new-password" />
        </a-form-item>
        <div v-if="error" class="form-error" role="alert">{{ error }}</div>
        <a-button type="primary" block html-type="submit" :loading="submitting">更新密码</a-button>
      </a-form>

      <div class="account-foot">
        <a-button type="text" danger @click="logout">
          <LogoutOutlined />
          退出登录
        </a-button>
        <span class="foot-hint">忘记密码请联系管理员重置</span>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.account {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.account-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--dt-line);
  border-radius: 12px;
  background: #fafbfd;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--dt-brand-soft);
  color: var(--dt-brand-hover);
  font-size: 16px;
  font-weight: 700;
}

.identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.identity strong {
  color: var(--dt-text);
  font-size: 13.6px;
}

.identity span {
  color: var(--dt-text-muted);
  font-size: 11.4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role-tag {
  padding: 2px 8px;
  border-radius: 999px;
  background: #fff;
  border: 1px solid var(--dt-line);
  color: var(--dt-text-muted);
  font-size: 10.4px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.info-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 12px;
  border-radius: 9px;
  background: #f9fafc;
}

.info-label {
  color: var(--dt-text-faint);
  font-size: 11px;
}

.info-value {
  color: var(--dt-text-secondary);
  font-size: 11.8px;
  font-weight: 600;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dt-text);
  font-size: 12.6px;
  font-weight: 650;
  margin-top: 2px;
}

.section-hint {
  margin: -10px 0 0;
  color: var(--dt-text-faint);
  font-size: 11px;
}

.form-error {
  margin-bottom: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--dt-danger-soft);
  color: var(--dt-danger-text);
  font-size: 11.6px;
}

.account-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--dt-line-soft);
}

.foot-hint {
  color: var(--dt-text-faint);
  font-size: 10.6px;
}

:deep(.ant-form-item) {
  margin-bottom: 12px;
}
</style>
