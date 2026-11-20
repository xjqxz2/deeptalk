<script setup lang="ts">
/**
 * 认证页布局：左侧 620 品牌区，右侧表单区；<1024 隐藏品牌区并保留小标。
 * 登录与注册共用，保证两页切换时视觉连续。
 */
import { CloudServerOutlined, SafetyCertificateOutlined, ThunderboltOutlined } from '@ant-design/icons-vue'
import BrandBlock from '@/components/base/BrandBlock.vue'

withDefaults(
  defineProps<{
    eyebrow: string
    title: string
    subtitle: string
    switchText: string
    switchLabel: string
    switchTo: string
  }>(),
  {},
)
</script>

<template>
  <div class="auth-shell">
    <section class="auth-brand" aria-hidden="true">
      <span class="blob blob-top" />
      <span class="blob blob-bottom" />
      <div class="brand-logo">
        <BrandBlock :size="34" subtitle="PERSONAL AI · MVP" />
      </div>
      <div class="brand-copy">
        <h2>让每一次对话，<br />都更接近答案。</h2>
        <p>DeepTalk 是你的个人 AI 工作台，<br />写代码、读文档、做分析，也能一起头脑风暴。</p>
        <ul class="features">
          <li>
            <span class="feature-icon"><ThunderboltOutlined /></span>
            深度思考与代码块原生支持
          </li>
          <li>
            <span class="feature-icon"><SafetyCertificateOutlined /></span>
            会话与用量按账号隔离
          </li>
          <li>
            <span class="feature-icon"><CloudServerOutlined /></span>
            管理员统一配置上游模型
          </li>
        </ul>
      </div>
      <footer class="brand-foot">
        <span>© 2026 DeepTalk</span>
        <span>隐私 · 条款</span>
      </footer>
    </section>

    <section class="auth-panel">
      <div class="panel-top">
        <BrandBlock class="compact-brand" :size="28" subtitle="" />
        <div class="switch">
          <span>{{ switchText }}</span>
          <router-link :to="switchTo">{{ switchLabel }}</router-link>
        </div>
      </div>

      <div class="form-wrap">
        <div class="form-inner">
          <p class="eyebrow">{{ eyebrow }}</p>
          <h1>{{ title }}</h1>
          <p class="subtitle">{{ subtitle }}</p>
          <slot />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth-shell {
  display: grid;
  grid-template-columns: 620px minmax(0, 1fr);
  min-height: 100vh;
  min-height: 100dvh;
  background: #fff;
}

.auth-brand {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 48px 60px 40px;
  background: linear-gradient(160deg, #f3f6ff 0%, #f8faff 55%, #ffffff 100%);
  border-right: 1px solid #eef1f7;
}

.blob {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
}

.blob-top {
  width: 560px;
  height: 560px;
  top: -230px;
  left: -120px;
  background: radial-gradient(circle at 40% 40%, #dfe8ff 0%, #eef3ff 45%, rgba(255, 255, 255, 0) 72%);
}

.blob-bottom {
  width: 620px;
  height: 620px;
  bottom: -330px;
  left: -160px;
  background: radial-gradient(circle at 55% 45%, #e6edff 0%, #f2f6ff 48%, rgba(255, 255, 255, 0) 74%);
}

.brand-logo {
  position: relative;
  z-index: 1;
}

.brand-copy {
  position: relative;
  z-index: 1;
  margin-top: auto;
  margin-bottom: auto;
  padding: 40px 0;
  max-width: 440px;
}

.brand-copy h2 {
  margin: 0 0 22px;
  color: var(--dt-text);
  font-size: 36px;
  line-height: 1.34;
  font-weight: 750;
  letter-spacing: -0.8px;
}

.brand-copy p {
  margin: 0 0 28px;
  color: var(--dt-text-muted);
  font-size: 13.6px;
  line-height: 23px;
}

.features {
  display: flex;
  flex-direction: column;
  gap: 13px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.features li {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
}

.feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  border: 1px solid #e4e9f5;
  background: rgba(255, 255, 255, 0.86);
  color: var(--dt-brand);
  font-size: 12px;
  box-shadow: 0 2px 6px rgba(77, 107, 254, 0.06);
}

.brand-foot {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 26px;
  border-top: 1px solid rgba(16, 24, 40, 0.06);
  color: var(--dt-text-faint);
  font-size: 10.4px;
}

.auth-panel {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  min-height: 100dvh;
  padding: calc(30px + env(safe-area-inset-top)) calc(54px + env(safe-area-inset-right))
    calc(36px + env(safe-area-inset-bottom)) calc(54px + env(safe-area-inset-left));
}

.panel-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 38px;
}

.compact-brand {
  display: none;
}

.switch {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dt-text-muted);
  font-size: 11.6px;
}

.switch a {
  color: var(--dt-brand);
  font-weight: 650;
}

.switch a:hover {
  color: var(--dt-brand-hover);
}

.form-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 0;
}

.form-inner {
  width: 100%;
  max-width: 380px;
}

.eyebrow {
  margin: 0 0 12px;
  color: var(--dt-text-faint);
  font-size: 9.6px;
  font-weight: 650;
  letter-spacing: 1.5px;
}

.form-inner h1 {
  margin: 0 0 10px;
  color: var(--dt-text);
  font-size: 30px;
  line-height: 1.3;
  font-weight: 750;
  letter-spacing: -0.6px;
}

.subtitle {
  margin: 0 0 30px;
  color: var(--dt-text-muted);
  font-size: 13px;
  line-height: 20px;
}

@media (max-width: 1240px) {
  .auth-shell {
    grid-template-columns: 520px minmax(0, 1fr);
  }

  .auth-brand {
    padding: 40px 42px 34px;
  }

  .brand-copy h2 {
    font-size: 32px;
  }
}

@media (max-width: 1024px) {
  .auth-shell {
    grid-template-columns: 1fr;
  }

  .auth-brand {
    display: none;
  }

  .auth-panel {
    padding: calc(24px + env(safe-area-inset-top)) calc(24px + env(safe-area-inset-right))
      calc(32px + env(safe-area-inset-bottom)) calc(24px + env(safe-area-inset-left));
    background: linear-gradient(180deg, #f7f9ff 0%, #ffffff 38%);
  }

  .compact-brand {
    display: inline-flex;
  }

  .form-wrap {
    padding: 24px 0;
  }
}

@media (max-width: 480px) {
  .auth-panel {
    padding: calc(20px + env(safe-area-inset-top)) calc(18px + env(safe-area-inset-right))
      calc(28px + env(safe-area-inset-bottom)) calc(18px + env(safe-area-inset-left));
  }

  .form-inner h1 {
    font-size: 26px;
  }

  .subtitle {
    margin-bottom: 22px;
  }
}
</style>
