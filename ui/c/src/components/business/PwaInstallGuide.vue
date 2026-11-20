<script setup lang="ts">
/**
 * 安装说明弹窗：iOS 走 Safari 分享菜单，Android 走浏览器安装菜单。
 * 这里只解释步骤，真正的系统安装弹窗由 usePwaInstall 触发。
 */
import {
  CheckOutlined,
  MoreOutlined,
  PlusSquareOutlined,
  ShareAltOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  ios: boolean
  android: boolean
  iosSafari: boolean
}>()
</script>

<template>
  <a-modal v-model:open="open" :footer="null" :width="420" centered title="安装 DeepTalk 到桌面">
    <div class="pwa-guide">
      <template v-if="props.ios">
        <p class="guide-lead">在 iPhone 或 iPad 上通过 Safari 添加到主屏幕，即可像 App 一样全屏使用。</p>
        <div v-if="!props.iosSafari" class="guide-warning">
          <WarningOutlined />
          当前浏览器不支持添加到主屏幕，请先用 Safari 打开本页。
        </div>
        <ol class="steps">
          <li>
            <span class="step-icon"><ShareAltOutlined /></span>
            <div class="step-copy">
              <strong>点击底部的分享按钮</strong>
              <span>在 Safari 工具栏找到「分享」图标。</span>
            </div>
          </li>
          <li>
            <span class="step-icon"><PlusSquareOutlined /></span>
            <div class="step-copy">
              <strong>选择「添加到主屏幕」</strong>
              <span>在分享菜单中向下滑动即可看到。</span>
            </div>
          </li>
          <li>
            <span class="step-icon"><CheckOutlined /></span>
            <div class="step-copy">
              <strong>点击右上角「添加」</strong>
              <span>完成后从桌面图标启动 DeepTalk。</span>
            </div>
          </li>
        </ol>
      </template>

      <template v-else>
        <p class="guide-lead">把 DeepTalk 安装到桌面，打开时没有浏览器地址栏，启动更快。</p>
        <ol class="steps">
          <li>
            <span class="step-icon"><MoreOutlined /></span>
            <div class="step-copy">
              <strong>打开浏览器右上角菜单</strong>
              <span>{{ props.android ? 'Chrome、Edge 等浏览器均可。' : '在浏览器菜单中找到安装入口。' }}</span>
            </div>
          </li>
          <li>
            <span class="step-icon"><PlusSquareOutlined /></span>
            <div class="step-copy">
              <strong>选择「安装应用」或「添加到主屏幕」</strong>
              <span>不同浏览器的文案可能略有差异。</span>
            </div>
          </li>
          <li>
            <span class="step-icon"><CheckOutlined /></span>
            <div class="step-copy">
              <strong>确认安装</strong>
              <span>完成后从桌面图标启动 DeepTalk。</span>
            </div>
          </li>
        </ol>
      </template>

      <p class="guide-tip">账号与历史对话会继续同步，不需要重新登录。</p>
      <a-button type="primary" block @click="open = false">我知道了</a-button>
    </div>
  </a-modal>
</template>

<style scoped>
.pwa-guide {
  padding-top: 2px;
}

.guide-lead {
  margin: 0 0 16px;
  color: var(--dt-text-secondary);
  font-size: 12.4px;
  line-height: 20px;
}

.guide-warning {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin: 0 0 14px;
  padding: 9px 11px;
  border-radius: 9px;
  background: var(--dt-warning-soft);
  color: var(--dt-warning-text);
  font-size: 11.6px;
  line-height: 18px;
}

.guide-warning :deep(.anticon) {
  margin-top: 2px;
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.steps li {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 10px 0;
  border-bottom: 1px solid var(--dt-line-soft);
}

.steps li:last-child {
  border-bottom: none;
}

.step-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  border-radius: 9px;
  background: var(--dt-brand-soft);
  color: var(--dt-brand);
  font-size: 14px;
}

.step-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding-top: 1px;
}

.step-copy strong {
  color: var(--dt-text);
  font-size: 12.4px;
  font-weight: 650;
  line-height: 18px;
}

.step-copy span {
  color: var(--dt-text-faint);
  font-size: 11.4px;
  line-height: 17px;
}

.guide-tip {
  margin: 14px 0;
  color: var(--dt-text-faint);
  font-size: 11px;
  text-align: center;
}
</style>
