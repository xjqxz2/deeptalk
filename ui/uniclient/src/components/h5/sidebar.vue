<template>
  <div class="sidebar-mask" @click="emit('close')" />
  <aside class="chat-sidebar mobile mobile-open" aria-label="对话历史">
    <div class="sidebar-head">
      <div class="brand-block">
        <BrandMark :size="30" :radius="8" />
        <span class="brand-text">小橘树洞</span>
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
          <span class="usage-title">{{ planName }}</span>
          <span class="usage-percent">本月 ¥{{ spend }} / ¥{{ spendLimit }}</span>
        </div>
        <div v-if="limits" class="usage-meta">
          <span class="usage-tokens">5 小时 / 本周 / 本月</span>
          <span class="usage-reset">{{ limits.requests_short }} / {{ limits.requests_week }} / {{ limits.requests_month }}</span>
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

const limits = computed(() => props.usage?.limits ?? null)
const planName = computed(() => props.usage?.limits?.plan?.name || '免费版')
const spend = computed(() => ((props.usage?.limits?.spend_cents ?? 0) / 100).toFixed(2))
const spendLimit = computed(() => ((props.usage?.limits?.spend_limit_cents ?? 0) / 100).toFixed(2))
</script>
