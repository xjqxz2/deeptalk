import { createSSRApp } from 'vue'
import App from './App.vue'

// H5 端复用 PWA 的视觉体系：同一个 theme.css、同一套组件样式
// #ifdef H5
import 'vant/lib/index.css'
import '../../shared/src/theme.css'
import './styles/pwa.css'
// #endif

export function createApp() {
  const app = createSSRApp(App)
  return { app }
}
