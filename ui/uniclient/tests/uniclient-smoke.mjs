/**
 * 小程序端（H5 调试构建）冒烟测试：
 * 登录页渲染、登录跳转、聊天页消息与思维链、发送走 WebSocket。
 */
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright-core'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '../../..')
const base = process.env.UNI_BASE || 'http://127.0.0.1:5273'
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

const user = { id: 1, email: 'u@example.com', nickname: 'U', role: 'user', plan: 'unlimited', quota_tokens: 0, used_tokens: 0, used_percent: 0, disabled: false, avatar_text: 'U' }
const conversation = { id: 7, public_id: 'u7', title: '小程序测试会话', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const messages = [
  { id: 1, role: 'user', content: '介绍一下你自己', reasoning: '', status: 'complete', model_name: '', input_tokens: 1, output_tokens: 0, duration_ms: 0, feedback: '', sources: [], attachments: [], created_at: '2026-11-20T10:00:00+08:00' },
  { id: 2, role: 'assistant', content: '**DeepTalk** 是一个对话工作台。\n\n- 支持联网\n- 支持代码\n\n```go\nfmt.Println("hi")\n```', reasoning: '用户想了解我。', status: 'complete', model_name: 'DeepSeek-V4', input_tokens: 1, output_tokens: 12, duration_ms: 3200, feedback: '', sources: [{ title: '示例来源', url: 'https://example.com', snippet: 's', viewed: true }], attachments: [], created_at: '2026-11-20T10:00:01+08:00' },
]

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'zh-CN' })
const errors = []
await context.route('**/api/v1/**', async (route) => {
  const p = new URL(route.request().url()).pathname
  const ok = (data) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data }) })
  if (p.endsWith('/c/auth/me')) return ok({ user })
  if (p.endsWith('/c/auth/login')) return ok({ user, token: 'test-token' })
  if (p.endsWith('/c/models')) return ok({ items: [{ id: 1, model_id: 'm', display_name: 'DeepSeek-V4', description: '', provider_name: 'D', context_window: 1000, is_default: true }] })
  if (p.endsWith('/c/conversations/u7')) return ok({ conversation, messages })
  if (p.endsWith('/c/conversations')) {
    if (route.request().method() === 'POST') return ok({ conversation: { ...conversation, id: 8, public_id: 'u8' } })
    return ok({ items: [conversation] })
  }
  if (p.endsWith('/c/usage')) return ok({ used_tokens: 0, quota_tokens: 0, used_percent: 0, reset_days: 30, plan: 'unlimited' })
  return ok(null)
})

// 拦截 WebSocket：实现 uni-h5 需要的 addEventListener / readyState / 常量，
// 并用假数据回放流式事件（真实 WS 由后端单测覆盖）
await context.addInitScript(() => {
  class FakeWS {
    static CONNECTING = 0
    static OPEN = 1
    static CLOSING = 2
    static CLOSED = 3
    constructor(url) {
      this.url = url
      this.readyState = 0
      this.binaryType = 'arraybuffer'
      this.listeners = {}
      window.__wsFrames = []
      setTimeout(() => {
        this.readyState = 1
        this.dispatch('open', {})
      }, 5)
    }
    get CONNECTING() { return 0 }
    get OPEN() { return 1 }
    get CLOSED() { return 3 }
    addEventListener(type, callback) {
      if (!this.listeners[type]) this.listeners[type] = []
      this.listeners[type].push(callback)
    }
    removeEventListener(type, callback) {
      this.listeners[type] = (this.listeners[type] || []).filter((fn) => fn !== callback)
    }
    dispatch(type, event) {
      ;(this.listeners[type] || []).forEach((fn) => fn(event))
    }
    send(payload) {
      window.__wsFrames.push(payload)
      const frame = JSON.parse(payload)
      if (frame.type !== 'stream') return
      const push = (type, data, delay) =>
        setTimeout(() => this.dispatch('message', { data: JSON.stringify({ type, payload: data }) }), delay)
      push('meta', { assistant_message_id: 99, model_name: 'DeepSeek-V4' }, 10)
      push('delta', { delta: '你好，' }, 30)
      push('delta', { delta: '我是 DeepTalk。' }, 60)
      push('done', { status: 'complete', title: '新会话标题' }, 90)
    }
    close() {
      this.readyState = 3
      this.dispatch('close', { code: 1000, reason: '' })
    }
  }
  window.WebSocket = FakeWS
})

const page = await context.newPage()
page.on('pageerror', (e) => errors.push(e.message))
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })

// 1. 登录页
await page.goto(`${base}/`, { waitUntil: 'networkidle' })
await page.waitForSelector('input', { timeout: 15000 })
const loginText = await page.locator('body').innerText()
if (!loginText.includes('欢迎回来')) throw new Error('login page not rendered')

// 2. 登录
await page.locator('input').nth(0).fill('u@example.com')
await page.locator('input').nth(1).fill('pass-1234')
await page.getByText('登录', { exact: true }).click()
await page.waitForSelector('.msg', { timeout: 15000 })

// 3. 历史消息渲染（Markdown 块、列表、代码、思维链）
const body = await page.locator('body').innerText()
for (const expect of ['DeepTalk', '支持联网', 'fmt.Println', '已思考 3 秒']) {
  if (!body.includes(expect)) throw new Error(`missing "${expect}" in chat page`)
}

// 4. 发送消息 → 走 WebSocket
await page.locator('textarea').fill('你好')
await page.getByText('发送', { exact: true }).click()
await page.waitForFunction(() => (window.__wsFrames || []).some((f) => JSON.parse(f).type === 'stream'), null, { timeout: 8000 })
await page.waitForFunction(() => document.body.innerText.includes('我是 DeepTalk'), null, { timeout: 8000 })

// 5. 抽屉
await page.getByText('☰').click()
await page.waitForSelector('.drawer')
if (!(await page.locator('.drawer').innerText()).includes('小程序测试会话')) throw new Error('drawer missing conversation')

await page.screenshot({ path: '/tmp/uniclient-chat.png' })
console.log(JSON.stringify({ ok: true, frames: await page.evaluate(() => window.__wsFrames.length), errors }))
await browser.close()
if (errors.length) process.exitCode = 1
