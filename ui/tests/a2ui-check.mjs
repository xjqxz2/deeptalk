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
const executablePath = process.env.PW_CHROMIUM || path.join(process.env.HOME || '', '.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell')

const user = { id: 1, email: 'a2ui@example.com', nickname: 'A2UI Tester', role: 'user', plan: 'unlimited', quota_tokens: 0, used_tokens: 0, used_percent: 0, disabled: false, created_at: '2026-01-01T00:00:00+08:00', last_login_at: null, last_login_city: '上海', last_login_ip: '127.0.0.1', avatar_text: 'A', conversation_count: 1, message_count: 2, quota_reset_days: 30 }
const model = { id: 1, model_id: 'deepseek-chat', display_name: 'DeepSeek-V4', description: '', provider_name: 'DeepSeek', context_window: 128000, is_default: true }
const conversation = { id: 9, public_id: 'a2ui-demo', title: 'A2UI 测试', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const a2ui = [
  { version: 'v0.9.1', createSurface: { surfaceId: 'a2ui-1', catalogId: 'https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json' } },
  { version: 'v0.9.1', updateComponents: { surfaceId: 'a2ui-1', components: [
    { id: 'root', component: 'Column', children: ['title', 'body'] },
    { id: 'title', component: 'Text', text: 'A2UI 测试标题', variant: 'h3' },
    { id: 'body', component: 'Text', text: '这是 AI 绘制的组件。' },
  ] } },
]
const messages = [
  { id: 1, role: 'user', content: '给我画一个卡片', reasoning: '', status: 'complete', model_name: '', input_tokens: 1, output_tokens: 0, duration_ms: 0, feedback: '', sources: [], attachments: [], created_at: '2026-11-20T10:00:00+08:00' },
  { id: 2, role: 'assistant', content: '界面已经生成。', reasoning: '', status: 'complete', model_name: 'DeepSeek-V4', input_tokens: 1, output_tokens: 1, duration_ms: 1000, feedback: '', sources: [], a2ui, attachments: [], created_at: '2026-11-20T10:00:01+08:00' },
]

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 900, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' })
await context.route('**/api/v1/**', async (route) => {
  const pathname = new URL(route.request().url()).pathname
  let data = null
  if (pathname.endsWith('/c/auth/me')) data = { user }
  else if (pathname.endsWith('/c/usage')) data = { used_tokens: 0, quota_tokens: 0, used_percent: 0, reset_days: 30, plan: 'unlimited' }
  else if (pathname.endsWith('/c/models')) data = { items: [model] }
  else if (pathname.endsWith('/c/conversations')) data = { items: [conversation] }
  else if (pathname.endsWith('/c/conversations/a2ui-demo')) data = { conversation, messages }
  await route.fulfill({ status: data ? 200 : 404, contentType: 'application/json', body: JSON.stringify({ code: data ? 0 : 404, msg: data ? 'ok' : 'not found', data }) })
})
const page = await context.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(e.message))
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
await page.goto(`${base}/chat/a2ui-demo`, { waitUntil: 'networkidle' })
await page.waitForSelector('a2ui-surface', { timeout: 15000 })
await page.waitForFunction(() => document.body.innerText.includes('A2UI 测试标题'), null, { timeout: 15000 })
const text = await page.locator('a2ui-surface').innerText()
await page.screenshot({ path: '/tmp/deeptalk-a2ui.png', fullPage: true })
console.log(JSON.stringify({ text, errors }))
await browser.close()
if (errors.length) process.exitCode = 1
