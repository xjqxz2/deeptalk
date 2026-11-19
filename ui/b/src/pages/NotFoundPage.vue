<script setup lang="ts">
/** B 端 404：引导回到概览。 */
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="not-found">
    <p class="code">404</p>
    <h1>页面不存在</h1>
    <p>你访问的后台地址不存在，可能已被移除。</p>
    <a-button type="primary" @click="router.push('/overview')">返回概览</a-button>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 100vh;
  background: var(--dt-bg);
  text-align: center;
}

.code {
  margin: 0;
  color: var(--dt-brand);
  font-family: var(--dt-mono);
  font-size: 48px;
  font-weight: 700;
}

h1 {
  margin: 0;
  font-size: 22px;
}

p {
  margin: 0 0 8px;
  color: var(--dt-text-muted);
  font-size: 12.6px;
}
</style>
