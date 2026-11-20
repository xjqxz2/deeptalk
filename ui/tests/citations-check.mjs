/** 引用上标与搜索 UI 回归检查。 */
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

const user = {
  id: 1, email: 'cite@example.com', nickname: 'Cite Tester', role: 'user', plan: 'unlimited',
  quota_tokens: 0, used_tokens: 0, used_percent: 0, disabled: false, created_at: '2026-01-01T00:00:00+08:00',
  last_login_at: null, last_login_city: '上海', last_login_ip: '127.0.0.1', avatar_text: 'C',
  conversation_count: 1, message_count: 2, quota_reset_days: 30,
}
const model = { id: 1, model_id: 'deepseek-chat', display_name: 'DeepSeek-V4', description: '', provider_name: 'DeepSeek', context_window: 128000, is_default: true }
const sources = [
  { title: 'DeepSeek 官方发布页', url: 'https://example.com/deepseek', snippet: 'V4 发布信息' },
  { title: '价格与额度说明', url: 'https://example.com/pricing', snippet: '价格信息' },
]
const conversation = { id: 7, public_id: 'cite-demo', title: '引用测试', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const messages = [
  { id: 1, role: 'user', content: 'DeepSeek V4 的价格是多少？', reasoning: '', status: 'complete', model_name: '', input_tokens: 1, output_tokens: 0, duration_ms: 0, feedback: '', sources: [], attachments: [], created_at: '2026-11-20T10:00:00+08:00' },
  { id: 2, role: 'assistant', content: 'DeepSeek V4 已发布[1]，价格可以参考官方说明[2]。', reasoning: '我需要先判断是否需要联网，再决定查哪个页面。', status: 'complete', model_name: 'DeepSeek-V4', input_tokens: 1, output_tokens: 1, duration_ms: 3200, feedback: '', sources, attachments: [], created_at: '2026-11-20T10:00:01+08:00' },
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
  else if (pathname.endsWith('/c/conversations/cite-demo')) data = { conversation, messages }
  await route.fulfill({ status: data ? 200 : 404, contentType: 'application/json', body: JSON.stringify({ code: data ? 0 : 404, msg: data ? 'ok' : 'not found', data }) })
})
const page = await context.newPage()
page.on('pageerror', (e) => { throw new Error(`pageerror: ${e.message}`) })
await page.goto(`${base}/chat/cite-demo`, { waitUntil: 'networkidle' })
await page.waitForSelector('.dt-cite')
const cites = page.locator('.dt-cite a')
if ((await cites.count()) !== 2) throw new Error(`citation count = ${await cites.count()}`)
if ((await cites.nth(0).getAttribute('href')) !== sources[0].url) throw new Error('first citation href mismatch')
if ((await cites.nth(1).getAttribute('title')) !== sources[1].title) throw new Error('citation title mismatch')
if ((await page.locator('.sources').count()) !== 0) throw new Error('source card should be removed')
if ((await page.locator('.searching').count()) !== 0) throw new Error('searching prompt should be removed')
if (!(await page.locator('.thinking-head').innerText()).includes('已思考')) throw new Error('thinking label mismatch')
await page.screenshot({ path: '/tmp/deeptalk-citations.png', fullPage: true })
console.log(JSON.stringify({ citations: await cites.count(), thinking: await page.locator('.thinking-head').innerText() }))
await browser.close()
