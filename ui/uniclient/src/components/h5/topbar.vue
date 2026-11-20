<template>
  <header class="chat-topbar">
    <div class="topbar-left">
      <Btn type="button" class="menu-btn" aria-label="展开侧栏" @click="emit('menu')">
        <MenuOutlined />
      </Btn>
      <h1 class="title" :title="title">{{ title }}</h1>
    </div>
    <div class="topbar-right">
      <Btn type="button" class="avatar" aria-label="账号菜单" @click="emit('account')">{{ initial }}</Btn>
    </div>
  </header>
</template>

<script setup>
/** H5 顶栏：结构与 class 名与 PWA 的 ChatTopbar 一致。 */
import { computed } from 'vue'
import { MenuOutlined } from '@ant-design/icons-vue'
import Btn from './btn.vue'

const props = defineProps({
  title: { type: String, default: '新对话' },
  initial: { type: String, default: 'U' },
})
const emit = defineEmits(['menu', 'account'])
</script>
