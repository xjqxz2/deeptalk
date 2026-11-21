<script setup lang="ts">
/** 应用根组件：统一主题与中文语言包，并跟随系统深浅色。 */
import { onBeforeUnmount, ref } from 'vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { antdTheme, darkAntdTheme, watchColorScheme } from '@config/theme'

const dark = ref(false)
const stop = watchColorScheme((value) => {
  dark.value = value
})
onBeforeUnmount(() => stop())
</script>

<template>
  <a-config-provider :theme="dark ? darkAntdTheme : antdTheme" :locale="zhCN">
    <router-view />
  </a-config-provider>
</template>
