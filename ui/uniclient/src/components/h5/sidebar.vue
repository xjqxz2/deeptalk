<template>
  <div class="sidebar-mask" @click="emit('close')" />
  <aside class="chat-sidebar mobile mobile-open" aria-label="对话历史">
    <div class="sidebar-head">
      <div class="brand-block">
        <BrandMark :size="30" :radius="8" />
        <span class="brand-text">DeepTalk</span>
      </div>
      <Btn type="button" class="collapse-btn" aria-label="关闭侧栏" @click="emit('close')">
        <MenuUnfoldOutlined />
      </Btn>
    </div>

    <div class="sidebar-body">
      <Btn type="button" class="new-chat" @click="emit('new')">
        <PlusOutlined />
        <span class="new-label">新建对话</span>
        <kbd>Ctrl K</kbd>
      </Btn>

      <div class="search-input-wrap">
        <SearchOutlined class="search-icon" />
        <NativeInput
          field-class="search-field"
          :model-value="keyword"
          placeholder="搜索对话"
          aria-label="搜索对话"
          @update:model-value="(value) => emit('search', value)"
        />
      </div>

      <div class="history">
        <template v-if="groups.length">
          <div v-for="group in groups" :key="group.label" class="history-group">
            <div class="group-label">{{ group.label }}</div>
            <div
              v-for="item in group.items"
              :key="item.id"
              class="history-item"
              :class="{ active: currentId === item.id }"
              role="button"
              tabindex="0"
              @click="emit('open', item)"
            >
              <MessageOutlined class="item-icon" />
              <span class="item-title" :title="item.title">{{ item.title }}</span>
              <Btn type="button" class="more" aria-label="会话操作" @click.stop="emit('actions', item)">
                <MoreOutlined />
              </Btn>
            </div>
          </div>
        </template>
        <div v-else class="history-empty">
          <span>{{ keyword ? '没有匹配的对话' : '还没有对话，开始第一段吧' }}</span>
        </div>
      </div>
    </div>

    <div class="sidebar-foot">
      <div class="usage-card">
        <div class="usage-head">
          <span class="usage-title">本月用量</span>
          <span class="usage-percent">{{ usageLabel }}</span>
        </div>
        <div class="usage-bar"><span class="usage-bar-fill" :style="{ width: usagePercent + '%' }" /></div>
        <div class="usage-meta">
          <span class="usage-tokens">{{ usageTokens }}</span>
          <span class="usage-reset">{{ usageReset }}</span>
        </div>
      </div>
      <Btn type="button" class="user-card" aria-label="账号菜单" @click="emit('account')">
        <span class="user-avatar">{{ initial }}</span>
        <span class="user-meta">
          <span class="user-name">{{ accountName }}</span>
          <span class="user-email">{{ accountMail }}</span>
        </span>
        <MoreOutlined class="user-more" />
      </Btn>
    </div>

  </aside>
</template>

<script setup>
/** H5 侧栏：结构与 class 名与 PWA 的 ChatSidebar 一致（手机态抽屉）。 */
import { computed } from 'vue'
import {
  MenuUnfoldOutlined,
  MessageOutlined,
  MoreOutlined,
  PlusOutlined,
  SearchOutlined,
} from '@ant-design/icons-vue'
import BrandMark from './brand-mark.vue'
import Btn from './btn.vue'
import NativeInput from './native-input.vue'

const props = defineProps({
  conversations: { type: Array, default: () => [] },
  currentId: { type: [Number, String], default: 0 },
  keyword: { type: String, default: '' },
  initial: { type: String, default: 'U' },
  accountName: { type: String, default: '' },
  accountMail: { type: String, default: '' },
  usage: { type: Object, default: null },
})
const emit = defineEmits(['close', 'new', 'search', 'open', 'actions', 'account'])

/** 简单分组：全部归到「今天」，与手机端一致，不再按日期拆多组。 */
const groups = computed(() => (props.conversations.length ? [{ label: '今天', items: props.conversations }] : []))

const usagePercent = computed(() => {
  const usage = props.usage
  if (!usage) return 0
  if (usage.quota_tokens > 0) return Math.min(100, Math.round((usage.used_tokens / usage.quota_tokens) * 100))
  return Math.min(100, usage.used_percent || 0)
})
const usageLabel = computed(() => (props.usage?.quota_tokens ? `${usagePercent.value}%` : '不限'))
const usageTokens = computed(() => {
  const usage = props.usage
  if (!usage) return '—'
  const fmt = (n) => (n >= 1000000 ? `${(n / 1000000).toFixed(1)}M` : n >= 1000 ? `${Math.round(n / 1000)}K` : String(n))
  return usage.quota_tokens ? `${fmt(usage.used_tokens)} / ${fmt(usage.quota_tokens)} tokens` : `${fmt(usage.used_tokens)} tokens`
})
const usageReset = computed(() => (props.usage?.reset_days ? `重置 ${props.usage.reset_days} 天` : '不设月度上限'))
</script>
