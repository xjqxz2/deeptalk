/** C 端应用入口：装配 Pinia、路由与 Ant Design Vue。 */
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import '@deeptalk/shared/style.css'
import './styles/global.css'
import App from './App.vue'
import router from './router'
import { registerServiceWorker } from './utils/pwa'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(Antd)
app.mount('#app')

// 只在生产构建注册；开发期不缓存，避免热更新被 Service Worker 拦截
registerServiceWorker()
