<script setup lang="ts">
/** 404 页面：保留品牌感，并提供返回工作台的主入口。 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { BrandMark } from '@deeptalk/shared'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const target = computed(() => (userStore.isLoggedIn ? '/' : '/login'))
</script>

<template>
  <div class="not-found">
    <BrandMark :size="44" :radius="13" />
    <p class="code">404</p>
    <h1>页面走丢了</h1>
    <p class="hint">你访问的地址不存在，或者已经被删除。</p>
    <a-button type="primary" @click="router.push(target)">返回工作台</a-button>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 100vh;
  padding: 32px;
  background: linear-gradient(180deg, var(--dt-brand-softer) 0%, var(--dt-surface) 52%);
  text-align: center;
}

.code {
  margin: 8px 0 0;
  color: var(--dt-brand);
  font-family: var(--dt-mono);
  font-size: 46px;
  font-weight: 700;
  letter-spacing: 2px;
}

h1 {
  margin: 0;
  color: var(--dt-text);
  font-size: 24px;
  font-weight: 730;
}

.hint {
  margin: 0 0 8px;
  color: var(--dt-text-muted);
  font-size: 13px;
}
</style>
