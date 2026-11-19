import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

/** 创建应用共用的 Vite 配置；adminBase 决定 B 端构建后的 /admin/ 前缀。 */
export function createViteConfig(options: { port: number; adminBase?: boolean }) {
  return defineConfig(({ mode }) => ({
    plugins: [vue()],
    // B 端生产构建使用 /admin/，开发期仍从根路径访问，便于本地调试
    base: options.adminBase ? (mode === 'production' ? '/admin/' : '/') : '/',
    resolve: {
      // 数组形式保证精确匹配优先：样式入口不能被 TS 入口的别名吞掉
      alias: [
        {
          find: '@deeptalk/shared/style.css',
          replacement: fileURLToPath(new URL('../../shared/src/theme.css', import.meta.url)),
        },
        { find: '@deeptalk/shared', replacement: fileURLToPath(new URL('../../shared/src/index.ts', import.meta.url)) },
        { find: '@config', replacement: fileURLToPath(new URL('.', import.meta.url)) },
        { find: '@', replacement: fileURLToPath(new URL('../src', import.meta.url)) },
      ],
    },
    server: {
      host: '0.0.0.0',
      port: options.port,
      strictPort: true,
      proxy: {
        // 开发期同源代理，Cookie 不跨域，避免 SameSite 问题
        '/api': { target: process.env.VITE_API_TARGET || 'http://127.0.0.1:9821', changeOrigin: false },
        '/healthz': { target: process.env.VITE_API_TARGET || 'http://127.0.0.1:9821', changeOrigin: false },
      },
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      chunkSizeWarningLimit: 900,
    },
  }))
}
