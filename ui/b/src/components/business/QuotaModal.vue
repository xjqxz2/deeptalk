<script setup lang="ts">
/** 额度调整弹窗：以 M tokens 为单位，校验不得低于已用量。 */
import { ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { ApiError, formatTokens } from '@deeptalk/shared'
import type { User } from '@deeptalk/shared'
import { updateQuota } from '@/services/admin'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ user: User | null }>()
const emit = defineEmits<{ (event: 'saved', user: User): void }>()

const saving = ref(false)
const error = ref('')
const quotaM = ref(4)

const maxQuota = 100

watch(open, (value) => {
  if (!value) return
  error.value = ''
  quotaM.value = props.user ? Math.max(1, Math.round(props.user.quota_tokens / 1_000_000)) : 4
})

/** 保存额度。 */
async function save(): Promise<void> {
  if (!props.user) return
  error.value = ''
  if (!Number.isFinite(quotaM.value) || quotaM.value < 1 || quotaM.value > maxQuota) {
    error.value = `额度需在 1M–${maxQuota}M 之间`
    return
  }
  if (quotaM.value * 1_000_000 < props.user.used_tokens) {
    error.value = `额度不能低于本月已用量 ${formatTokens(props.user.used_tokens)}`
    return
  }
  saving.value = true
  try {
    const data = await updateQuota(props.user.id, Math.round(quotaM.value * 1_000_000))
    emit('saved', data.user)
    message.success('额度已更新')
    open.value = false
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '保存失败，请稍后重试'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <a-modal v-model:open="open" title="调整额度" :width="420" :footer="null" destroy-on-close>
    <div class="quota">
      <div class="summary">
        <div>
          <span>当前已用</span>
          <strong>{{ formatTokens(user?.used_tokens ?? 0) }}</strong>
        </div>
        <div>
          <span>当前额度</span>
          <strong>{{ formatTokens(user?.quota_tokens ?? 0) }}</strong>
        </div>
      </div>

      <label class="field-label" for="quota-input">本月额度（M tokens）</label>
      <a-input-number
        id="quota-input"
        v-model:value="quotaM"
        :min="1"
        :max="maxQuota"
        :step="1"
        style="width: 100%"
      />
      <p class="hint">1M = 1,000,000 tokens；额度用尽后仍可登录，但不能发起新对话。</p>

      <div v-if="error" class="form-error" role="alert">{{ error }}</div>

      <div class="actions">
        <a-button @click="open = false">取消</a-button>
        <a-button type="primary" :loading="saving" @click="save">保存</a-button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.quota {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.summary {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 4px;
}

.summary div {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 10px 12px;
  border-radius: 9px;
  background: #fafbfd;
  border: 1px solid var(--dt-line-soft);
}

.summary span {
  color: var(--dt-text-faint);
  font-size: 9.8px;
}

.summary strong {
  color: var(--dt-text);
  font-size: 15px;
  font-weight: 680;
}

.field-label {
  color: var(--dt-text-secondary);
  font-size: 11.4px;
  font-weight: 600;
}

.hint {
  margin: 0;
  color: var(--dt-text-faint);
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
  margin-top: 8px;
}
</style>
