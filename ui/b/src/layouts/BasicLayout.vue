<script setup lang="ts">
/**
 * 后台主壳：240 侧栏 + 60 顶栏 + 内容区。
 * 负责外壳数据加载、移动端抽屉、管理员指南弹窗。
 */
import { onBeforeUnmount, onMounted } from 'vue'
import { BookOutlined, KeyOutlined, SafetyCertificateOutlined, SwapOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/stores/app'
import { useAdminStore } from '@/stores/admin'
import AdminSidebar from '@/components/layout/AdminSidebar.vue'
import AdminTopbar from '@/components/layout/AdminTopbar.vue'

const app = useAppStore()
const admin = useAdminStore()

/** 同步断点。 */
function onResize(): void {
  app.syncViewport(window.innerWidth)
}

onMounted(() => {
  app.syncViewport(window.innerWidth)
  window.addEventListener('resize', onResize)
  void admin.loadShell()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
})
</script>

<template>
  <div class="admin-shell">
    <AdminSidebar :class="{ 'mobile-open': app.mobileSidebarOpen }" />
    <button
      v-if="app.isMobile && app.mobileSidebarOpen"
      type="button"
      class="sidebar-backdrop"
      aria-label="关闭导航"
      @click="app.closeSidebar"
    />
    <div class="admin-main">
      <AdminTopbar />
      <main class="admin-content">
        <router-view />
      </main>
    </div>

    <a-modal v-model:open="app.guideOpen" title="管理员指南" :footer="null" :width="560">
      <div class="guide">
        <section>
          <h3><KeyOutlined /> 密钥托管</h3>
          <p>
            上游 API Key 使用 AES-GCM 加密后入库，接口只返回掩码。浏览器永远拿不到明文；
            更换 Key 后需要重新测试连通性。
          </p>
        </section>
        <section>
          <h3><SwapOutlined /> 模型路由</h3>
          <p>
            默认模型失败或超时时，仅触发一次兜底模型；MVP 不做跨厂商自动重试，避免重复计费。
            停用的模型不会下发给 C 端。
          </p>
        </section>
        <section>
          <h3><SafetyCertificateOutlined /> 安全边界</h3>
          <p>
            后台不读取用户对话正文，CSV 只导出账号字段；限制登录、重置密码会立即撤销该用户全部会话。
          </p>
        </section>
        <section>
          <h3><BookOutlined /> 版本</h3>
          <p>DeepTalk Admin v1.0.0 · 单实例 SQLite 部署。更多配置项见 SPEC.md。</p>
        </section>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.admin-shell {
  display: flex;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  background: var(--dt-bg);
}

.admin-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  height: 100%;
}

.admin-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 32px 40px;
}

.sidebar-backdrop {
  position: fixed;
  inset: 0;
  z-index: 35;
  border: none;
  background: rgba(16, 24, 40, 0.32);
  cursor: pointer;
}

.guide {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.guide h3 {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 0 6px;
  color: var(--dt-text);
  font-size: 13.4px;
  font-weight: 650;
}

.guide h3 :deep(svg) {
  color: var(--dt-brand);
}

.guide p {
  margin: 0;
  color: var(--dt-text-muted);
  font-size: 12.4px;
  line-height: 21px;
}

@media (max-width: 1024px) {
  .admin-content {
    padding: 18px 16px 32px;
  }
}
</style>
