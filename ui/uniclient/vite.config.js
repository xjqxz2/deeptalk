import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

// H5 调试时把接口代理到后端，避免跨域；生产构建仍直连 BASE_URL
const BACKEND = process.env.VITE_PROXY_TARGET || 'https://deeptalk.peon.top'

export default defineConfig({
  plugins: [uni()],
  server: {
    host: '0.0.0.0',
    port: 5273,
    proxy: {
      '/api': {
        target: BACKEND,
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
