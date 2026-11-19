<script setup lang="ts">
/** 应用根组件：统一主题与中文语言包。 */
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { antdTheme } from '@config/theme'
</script>

<template>
  <a-config-provider :theme="antdTheme" :locale="zhCN">
    <router-view />
  </a-config-provider>
</template>
