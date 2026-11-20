import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers'
import { defineConfig, type Plugin } from 'vite'

/**
 * 给 Service Worker 注入构建版本。
 * public/ 是原样拷贝，若不替换 __PWA_VERSION__，每次发版的缓存名都一样，旧资源无法淘汰。
 */
function pwaServiceWorker(): Plugin {
  let root = ''
  let outDir = ''

  return {
    name: 'deeptalk-pwa-service-worker',
    apply: 'build',
    configResolved(config) {
      root = config.root
      outDir = config.build.outDir
    },
    async closeBundle() {
      const source = resolve(root, 'public/sw.js')
      const target = resolve(root, outDir, 'sw.js')
      const template = await readFile(source, 'utf8')
      if (!template.includes('__PWA_VERSION__')) {
        throw new Error('public/sw.js 缺少 __PWA_VERSION__ 占位符')
      }
      const version = Date.now().toString(36)
      await writeFile(target, template.replaceAll('__PWA_VERSION__', version))
    },
  }
}

/** 创建应用共用的 Vite 配置；adminBase 决定 B 端构建后的 /admin/ 前缀。 */
export function createViteConfig(options: { port: number; adminBase?: boolean; pwa?: boolean; antdOnDemand?: boolean }) {
  return defineConfig(({ mode }) => ({
    plugins: [
      vue(),
      // 按需注册 antd 组件：整包注册会把全部组件打进首屏，体积翻倍。
      // antd v4 的样式是 CSS-in-JS，不需要按组件引入样式文件。
      ...(options.antdOnDemand
        ? [
            Components({
              dts: false,
              resolvers: [AntDesignVueResolver({ importStyle: false, resolveIcons: false })],
            }),
          ]
        : []),
      ...(options.pwa ? [pwaServiceWorker()] : []),
    ],
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
      rollupOptions: {
        output: {
          // 把长期不变的依赖拆成独立 chunk：发版后浏览器只重新下载业务代码，
          // 同时避免所有依赖挤在一个 2MB+ 的首屏文件里。
          manualChunks(id) {
            if (!id.includes('node_modules')) return undefined
            if (id.includes('highlight.js')) return 'vendor-hljs'
            // icons 与 antd 有循环依赖，必须进同一个 chunk，否则初始化顺序会导致
            // 图标组件拿不到主题 token（Cannot read properties of undefined）
            if (id.includes('ant-design-vue') || id.includes('@ant-design/icons-vue')) return 'vendor-antd'
            if (
              id.includes('markdown-it') ||
              id.includes('dompurify') ||
              id.includes('linkify-it') ||
              id.includes('mdurl') ||
              id.includes('uc.micro')
            ) {
              return 'vendor-markdown'
            }
            if (
              id.includes('/vue/') ||
              id.includes('@vue/') ||
              id.includes('vue-router') ||
              id.includes('pinia')
            ) {
              return 'vendor-vue'
            }
            return undefined
          },
        },
      },
    },
  }))
}
