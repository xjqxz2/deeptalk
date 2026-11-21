/**
 * 小橘树洞 C 端 Service Worker。
 * 只缓存公开的静态资源与离线页，不缓存 /api（对话、SSE、账号数据永不落盘到 Cache Storage）。
 * 构建时由 config/vite.ts 注入本次构建版本。
 */
const VERSION = '__PWA_VERSION__'
const CACHE_PREFIX = 'deeptalk-c-'
const CACHE_NAME = `${CACHE_PREFIX}${VERSION}`
const OFFLINE_URL = '/offline.html'

/** 安装时直接缓存的最小离线资源。 */
const PRECACHE_URLS = [
  OFFLINE_URL,
  '/manifest.webmanifest',
  '/icons/favicon-64.png',
  '/icons/favicon-192.png',
  '/icons/icon-192.png',
  '/icons/shark-icon.png',
  '/icons/icon-maskable-192.png',
  '/icons/icon-maskable-512.png',
  '/icons/apple-touch-icon.png',
]

/** 逐个缓存，单个资源失败不能导致整个 Service Worker 安装失败。 */
async function precache(cache, urls) {
  await Promise.allSettled(
    urls.map(async (url) => {
      const response = await fetch(new Request(url, { cache: 'reload' }))
      if (response.ok) await cache.put(url, response)
    }),
  )
}

/** 预缓存首屏 JS/CSS，减少再次打开时的等待；解析失败不影响安装。 */
async function precacheBuildAssets(cache) {
  try {
    const response = await fetch(new Request('/', { cache: 'reload' }))
    if (!response.ok) return
    const html = await response.text()
    const assets = new Set()
    const pattern = /(?:src|href)="(\/assets\/[^"]+)"/g
    let match = pattern.exec(html)
    while (match) {
      assets.add(match[1])
      match = pattern.exec(html)
    }
    await precache(cache, [...assets])
  } catch {
    // 首屏资源预缓存是增强项，失败时交给运行时缓存
  }
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE_NAME)
      await precache(cache, PRECACHE_URLS)
      await precacheBuildAssets(cache)
      // 新版本立即接管；旧缓存保留一代，避免旧页面懒加载 chunk 失效
      await self.skipWaiting()
    })(),
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys()
      const appCaches = keys.filter((key) => key.startsWith(CACHE_PREFIX) && key !== CACHE_NAME)
      const keep = new Set([CACHE_NAME, ...appCaches.slice(-1)])
      await Promise.all(
        keys.filter((key) => key.startsWith(CACHE_PREFIX) && !keep.has(key)).map((key) => caches.delete(key)),
      )
      await self.clients.claim()
    })(),
  )
})

/** 页面导航优先走网络，断网时返回离线说明页。 */
async function networkFirstNavigation(request) {
  try {
    return await fetch(request)
  } catch {
    const cached = await caches.match(OFFLINE_URL)
    if (cached) return cached
    return new Response('网络连接已断开，请恢复网络后重试。', {
      status: 503,
      headers: { 'Content-Type': 'text/plain; charset=utf-8' },
    })
  }
}

/** 带内容哈希的构建产物可以长期缓存，未命中时再请求网络并写回。 */
async function cacheFirst(request) {
  const cached = await caches.match(request)
  if (cached) return cached
  const response = await fetch(request)
  if (response.ok) {
    const cache = await caches.open(CACHE_NAME)
    await cache.put(request, response.clone())
  }
  return response
}

/** manifest 使用 stale-while-revalidate，更新后下次打开生效。 */
async function staleWhileRevalidate(request) {
  const cached = await caches.match(request)
  const network = fetch(request)
    .then(async (response) => {
      if (response.ok) {
        const cache = await caches.open(CACHE_NAME)
        await cache.put(request, response.clone())
      }
      return response
    })
    .catch(() => null)
  return cached || (await network) || Response.error()
}

self.addEventListener('fetch', (event) => {
  const request = event.request
  if (request.method !== 'GET') return

  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return
  // 管理后台与接口不进入 C 端缓存
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/admin')) return

  if (request.mode === 'navigate') {
    event.respondWith(networkFirstNavigation(request))
    return
  }

  if (url.pathname.startsWith('/assets/') || url.pathname.startsWith('/icons/')) {
    event.respondWith(cacheFirst(request))
    return
  }

  if (url.pathname === '/manifest.webmanifest') {
    event.respondWith(staleWhileRevalidate(request))
  }
})

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') self.skipWaiting()
})
