/**
 * C 端 PWA 冒烟检查：manifest、图标、Service Worker、离线页与安装入口。
 * 运行：node tests/pwa-check.mjs（需要先 pnpm --filter @deeptalk/c build && pnpm --filter @deeptalk/c preview）
 */
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright-core'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '../..')
const base = process.env.BASE_URL || 'http://127.0.0.1:4173'
const deps = path.join(root, '.browser-deps')
process.env.LD_LIBRARY_PATH = [
  path.join(deps, 'root/usr/lib/x86_64-linux-gnu'),
  path.join(deps, 'root/usr/lib'),
  path.join(deps, 'root/lib/x86_64-linux-gnu'),
  process.env.LD_LIBRARY_PATH || '',
].join(':')
process.env.FONTCONFIG_FILE = path.join(deps, 'fonts.conf')
process.env.FONTCONFIG_PATH = deps

const executablePath =
  process.env.PW_CHROMIUM ||
  path.join(process.env.HOME || '', '.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell')

const results = []
const errors = []
const check = (name, condition) => {
  results.push({ name, ok: Boolean(condition) })
  if (!condition) throw new Error(`断言失败：${name}`)
}

const user = {
  id: 1,
  email: 'pwa@example.com',
  nickname: 'PWA Tester',
  role: 'user',
  plan: 'unlimited',
  quota_tokens: 0,
  used_tokens: 0,
  used_percent: 0,
  disabled: false,
  created_at: '2026-01-01T00:00:00+08:00',
  last_login_at: null,
  last_login_city: '上海',
  last_login_ip: '127.0.0.1',
  avatar_text: 'P',
  conversation_count: 0,
  message_count: 0,
  quota_reset_days: 30,
}
const model = {
  id: 1,
  model_id: 'deepseek-chat',
  display_name: 'DeepSeek-V4',
  description: '',
  provider_name: 'DeepSeek',
  context_window: 128000,
  is_default: true,
}

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
const context = await browser.newContext({
  viewport: { width: 390, height: 844 },
  deviceScaleFactor: 3,
  isMobile: true,
  hasTouch: true,
  locale: 'zh-CN',
  timezoneId: 'Asia/Shanghai',
  userAgent:
    'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1',
  serviceWorkers: 'allow',
})

await context.route('**/api/v1/**', async (route) => {
  const pathname = new URL(route.request().url()).pathname
  let data = null
  if (pathname.endsWith('/c/auth/me')) data = { user }
  else if (pathname.endsWith('/c/usage')) data = { used_tokens: 0, quota_tokens: 0, used_percent: 0, reset_days: 30, plan: 'unlimited' }
  else if (pathname.endsWith('/c/models')) data = { items: [model] }
  else if (pathname.endsWith('/c/conversations')) data = { items: [] }
  await route.fulfill({
    status: data ? 200 : 404,
    contentType: 'application/json',
    body: JSON.stringify({ code: data ? 0 : 404, msg: data ? 'ok' : 'not found', data }),
  })
})

const page = await context.newPage()
page.on('pageerror', (e) => errors.push(`pageerror: ${e.message}`))
page.on('console', (m) => {
  if (m.type() === 'error') errors.push(`console: ${m.text()}`)
})

await page.goto(`${base}/`, { waitUntil: 'networkidle' })
await page.waitForSelector('.chat-empty')

check('空态副标题已移除', (await page.locator('.hero-sub').count()) === 0)
check('空态标题保留', (await page.locator('.empty-hero h1').innerText()).includes('DeepTalk'))
check('manifest 已声明', (await page.locator('link[rel="manifest"]').getAttribute('href')) === '/manifest.webmanifest')
check('apple-touch-icon 已声明', (await page.locator('link[rel="apple-touch-icon"]').count()) === 1)

const manifest = await page.evaluate(async () => (await fetch('/manifest.webmanifest')).json())
check('manifest 为 standalone', manifest.display === 'standalone')
check('manifest 包含 192/512 图标', manifest.icons.some((i) => i.sizes === '192x192') && manifest.icons.some((i) => i.sizes === '512x512'))
check('manifest 包含 maskable 图标', manifest.icons.some((i) => i.purpose === 'maskable'))
for (const icon of manifest.icons) {
  const response = await page.request.get(`${base}${icon.src}`)
  check(`图标可访问 ${icon.src}`, response.ok())
}
const appleIcon = await page.request.get(`${base}/icons/apple-touch-icon.png`)
check('Apple Touch Icon 可访问', appleIcon.ok())

const cdp = await context.newCDPSession(page)
const appManifest = await cdp.send('Page.getAppManifest')
check('浏览器识别 manifest', Boolean(appManifest.url))
const installability = await cdp.send('Page.getInstallabilityErrors')
check(
  '无安装性错误',
  installability.installabilityErrors.length === 0 ||
    installability.installabilityErrors.every((item) => !item.errorId || item.errorId === 'no-icon-available'),
)

await page.evaluate(() => navigator.serviceWorker.ready)
await page.waitForFunction(() => Boolean(navigator.serviceWorker.controller), null, { timeout: 10000 })
const cacheState = await page.evaluate(async () => {
  const names = await caches.keys()
  const name = names.find((item) => item.startsWith('deeptalk-c-'))
  if (!name) return { names, paths: [] }
  const cache = await caches.open(name)
  const keys = await cache.keys()
  return { names, paths: keys.map((request) => new URL(request.url).pathname) }
})
check('Service Worker 已建立缓存', cacheState.paths.includes('/offline.html'))
check('Service Worker 缓存图标', cacheState.paths.some((item) => item.includes('icon-192')))
check('Service Worker 不缓存 API', !cacheState.paths.some((item) => item.startsWith('/api/')))

// iOS 安装入口与说明
await page.locator('.avatar').click()
await page.getByRole('menuitem', { name: '安装到桌面' }).click()
await page.waitForSelector('.ant-modal-content')
check('iOS 安装说明出现', (await page.locator('.ant-modal-content').innerText()).includes('添加到主屏幕'))
check('iOS 安装说明包含 Safari', (await page.locator('.ant-modal-content').innerText()).includes('Safari'))
await page.screenshot({ path: path.join(here, '../../docs/qa/pwa-ios-install-guide.png') })
await page.getByRole('button', { name: '我知道了' }).click()
await page.waitForTimeout(450)

await page.screenshot({ path: path.join(here, '../../docs/qa/pwa-ios-empty.png') })

// 断网时导航到任意会话，应返回离线页
await context.setOffline(true)
await page.goto(`${base}/chat/offline-check`, { waitUntil: 'domcontentloaded' })
await page.waitForFunction(() => document.body.innerText.includes('网络连接已断开'), null, { timeout: 10000 })
check('离线页可显示', (await page.locator('h1').innerText()).includes('网络连接已断开'))
await context.setOffline(false)

await browser.close()

console.log(JSON.stringify({ results, errors, cacheState }, null, 2))
if (errors.length) {
  console.error(errors.join('\n'))
  process.exitCode = 1
}
