/**
 * 并行对话与检索活动提示回归：
 * 1. 会话 A 生成中可新建会话，A 不被中断；
 * 2. 侧栏显示 A 仍在生成；
 * 3. 回到 A 能看到持续输出；
 * 4. 思考面板展示“正在搜索/正在阅读网页”等活动提示。
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
const executablePath = process.env.PW_CHROMIUM || path.join(process.env.HOME || '', '.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell')

const user = { id: 1, email: 'p@example.com', nickname: 'P', role: 'user', plan: 'unlimited', quota_tokens: 0, used_tokens: 0, used_percent: 0, disabled: false, created_at: '2026-01-01T00:00:00+08:00', last_login_at: null, last_login_city: '上海', last_login_ip: '127.0.0.1', avatar_text: 'P', conversation_count: 2, message_count: 2, quota_reset_days: 30 }
const model = { id: 1, model_id: 'deepseek-chat', display_name: 'DeepSeek-V4', description: '', provider_name: 'DeepSeek', context_window: 128000, is_default: true }
const convA = { id: 101, public_id: 'conv-a', title: '会话 A', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 0 }
const convB = { id: 102, public_id: 'conv-b', title: '新对话', model_id: 1, updated_at: '2026-11-20T10:00:01+08:00', message_count: 0 }

const sse = (events) => events.map(([event, data]) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`).join('')

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 1100, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' })
let streamStarted = 0
await context.route('**/api/v1/**', async (route) => {
  const url = new URL(route.request().url())
  const pathname = url.pathname
  const fulfill = (data) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data }) })
  if (pathname.endsWith('/c/auth/me')) return fulfill({ user })
  if (pathname.endsWith('/c/usage')) return fulfill({ used_tokens: 0, quota_tokens: 0, used_percent: 0, reset_days: 30, plan: 'unlimited' })
  if (pathname.endsWith('/c/models')) return fulfill({ items: [model] })
  if (pathname.endsWith('/c/conversations')) return fulfill({ items: [convB, convA] })
  if (pathname.endsWith('/c/conversations/conv-a')) {
    return fulfill({
      conversation: convA,
      messages: [{ id: 1, role: 'user', content: '查一下今天的新闻', reasoning: '', status: 'complete', model_name: '', input_tokens: 1, output_tokens: 0, duration_ms: 0, feedback: '', sources: [], attachments: [], created_at: '2026-11-20T10:00:00+08:00' }],
    })
  }
  if (pathname.endsWith('/c/conversations/conv-b')) return fulfill({ conversation: convB, messages: [] })
  if (pathname.endsWith('/c/conversations/101/stream')) {
    streamStarted += 1
    // 交给本地慢速 SSE 服务，保证能观察到流式过程
    return route.continue({ url: 'http://127.0.0.1:4199/stream' })
  }
  return route.fulfill({ status: 404, contentType: 'application/json', body: JSON.stringify({ code: 404, msg: 'not found', data: null }) })
})

const page = await context.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(e.message))
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })

await page.goto(`${base}/chat/conv-a`, { waitUntil: 'networkidle' })
await page.fill('.composer-input', '查一下今天的新闻')
await page.press('.composer-input', 'Enter')
await page.waitForFunction(() => document.body.innerText.includes('正在阅读网页'), null, { timeout: 10000 })
const activityText = await page.locator('.thinking-activity').innerText()
if (!activityText.includes('今日头条')) throw new Error(`activity detail missing: ${activityText}`)

// 生成中允许新建对话，且不打断 A
await page.locator('.new-chat').click()
await page.waitForURL(`${base}/`)
await page.waitForSelector('.chat-empty')
if (streamStarted !== 1) throw new Error('stream should start exactly once')

// 侧栏应显示 A 仍在生成
await page.waitForFunction(() => document.querySelectorAll('.history-item .anticon-loading').length === 1, null, { timeout: 5000 })
const badge = await page.locator('.history-item').filter({ hasText: '会话 A' }).first().locator('.anticon-loading').count()
if (badge !== 1) throw new Error('sidebar should show generating badge for 会话 A')

// 回到 A 应看到继续输出
await page.locator('.history-item').filter({ hasText: '会话 A' }).first().click()
await page.waitForURL(`${base}/chat/conv-a`)
await page.waitForFunction(() => document.body.innerText.includes('今天的主要新闻是'), null, { timeout: 10000 })

await page.screenshot({ path: '/tmp/deeptalk-parallel.png', fullPage: true })
console.log(JSON.stringify({ activityText, streamStarted, errors }))
await browser.close()
if (errors.length) process.exitCode = 1
