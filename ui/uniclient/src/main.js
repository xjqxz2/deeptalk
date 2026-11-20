import { createSSRApp } from 'vue'
import App from './App.vue'

// Vant 只在 H5 端引入：小程序与 nvue 端没有 DOM，无法使用 Vant 组件
// #ifdef H5
import 'vant/lib/index.css'
// #endif

export function createApp() {
  const app = createSSRApp(App)
  return { app }
}
