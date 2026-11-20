/**
 * 分享功能回归：
 * 1. 会话页可创建分享并显示链接；
 * 2. 分享页只读，没有输入框与回复入口；
 * 3. 撤销后分享页显示失效；
 * 4. 查看了网页的链接会被保留并展示。
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

const user = { id: 1, email: 's@example.com', nickname: '分享者', role: 'user', plan: 'unlimited', quota_tokens: 0, used_tokens: 0, used_percent: 0, disabled: false, created_at: '2026-01-01T00:00:00+08:00', last_login_at: null, last_login_city: '上海', last_login_ip: '127.0.0.1', avatar_text: 'S', conversation_count: 1, message_count: 2, quota_reset_days: 30 }
const model = { id: 1, model_id: 'deepseek-chat', display_name: 'DeepSeek-V4', description: '', provider_name: 'DeepSeek', context_window: 128000, is_default: true }
const conversation = { id: 55, public_id: 'share-55', title: '分享测试会话', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const sources = [
  { title: '今日头条', url: 'https://example.com/news', snippet: '新闻摘要', viewed: true },
  { title: '另一个搜索结果', url: 'https://example.com/other', snippet: '仅摘要' },
]
const messages = [
  { id: 1, role: 'user', content: '今天有什么新闻？', reasoning: '', status: 'complete', model_name: '', input_tokens: 1, output_tokens: 0, duration_ms: 0, feedback: '', sources: [], attachments: [], created_at: '2026-11-20T10:00:00+08:00' },
  { id: 2, role: 'assistant', content: '今天的新闻主要是这些[1]。', reasoning: '需要联网。', status: 'complete', model_name: 'DeepSeek-V4', input_tokens: 1, output_tokens: 1, duration_ms: 1000, feedback: '', sources, attachments: [], created_at: '2026-11-20T10:00:01+08:00' },
]

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 900, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' })
let shareToken = ''
const errors = []
await context.route('**/api/v1/**', async (route) => {
  const pathname = new URL(route.request().url()).pathname
  const method = route.request().method()
  const ok = (data) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data }) })
  if (pathname.endsWith('/c/auth/me')) return ok({ user })
  if (pathname.endsWith('/c/usage')) return ok({ used_tokens: 0, quota_tokens: 0, used_percent: 0, reset_days: 30, plan: 'unlimited' })
  if (pathname.endsWith('/c/models')) return ok({ items: [model] })
  if (pathname.endsWith('/c/conversations')) return ok({ items: [conversation] })
  if (pathname.endsWith('/c/conversations/share-55')) return ok({ conversation, messages })
  if (pathname.endsWith('/c/conversations/55/share')) {
    if (method === 'POST') {
      shareToken = 'tok-abcdef'
      return ok({ share: { token: shareToken, created_at: '2026-11-20T10:05:00+08:00' } })
    }
    if (method === 'DELETE') {
      shareToken = ''
      return ok(null)
    }
    return ok({ share: null })
  }
  if (pathname.includes('/c/public/shares/')) {
    const token = pathname.split('/').pop()
    if (token !== shareToken || !shareToken) {
      return route.fulfill({ status: 404, contentType: 'application/json', body: JSON.stringify({ code: 9002, msg: '分享不存在或已失效', data: null }) })
    }
    return ok({ title: conversation.title, owner: '分享者', shared_at: '2026-11-20T10:05:00+08:00', messages })
  }
  return route.fulfill({ status: 404, contentType: 'application/json', body: JSON.stringify({ code: 404, msg: 'not found', data: null }) })
})

const page = await context.newPage()
page.on('pageerror', (e) => errors.push(e.message))
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })

// 1. 看过的网页融合进思维链，且不再显示数量
await page.goto(`${base}/chat/share-55`, { waitUntil: 'networkidle' })
await page.waitForSelector('.thinking')
const thinkingText = await page.locator('.thinking').innerText()
if (thinkingText.includes('查看了')) throw new Error(`should not show viewed count: ${thinkingText}`)
if (!(await page.locator('.thinking').evaluate((el) => el.classList.contains('expanded')))) {
  await page.locator('.thinking-head').click()
}
await page.waitForSelector('.thinking-source')
const refLinks = await page.locator('.thinking-source-title').allInnerTexts()
if (refLinks.length !== 1 || refLinks[0] !== '今日头条') throw new Error(`viewed links wrong: ${refLinks}`)
if ((await page.locator('.refs').count()) !== 0) throw new Error('standalone refs block should be gone')
const refsText = 'thinking'

// 2. 创建分享
await page.locator('button[aria-label="分享对话"]').click()
await page.getByRole('button', { name: '创建分享链接' }).click()
await page.waitForSelector('.share-input')
const link = await page.locator('.share-input').inputValue()
if (!link.endsWith('/s/tok-abcdef')) throw new Error(`unexpected share link: ${link}`)

// 3. 分享页只读
const shared = await context.newPage()
shared.on('pageerror', (e) => errors.push(`share: ${e.message}`))
await shared.goto(link, { waitUntil: 'networkidle' })
await shared.waitForSelector('.share-title-block')
if ((await shared.locator('.composer').count()) !== 0) throw new Error('share page must not have composer')
if ((await shared.getByRole('button', { name: '重新生成' }).count()) !== 0) throw new Error('share page must not allow regenerate')
if ((await shared.getByRole('button', { name: '赞' }).count()) !== 0) throw new Error('share page must not allow feedback')
if (!(await shared.locator('.share-foot').innerText()).includes('只读分享')) throw new Error('missing readonly hint')
await shared.screenshot({ path: '/tmp/deeptalk-share.png', fullPage: true })

// 4. 撤销
await page.getByRole('button', { name: '取消分享' }).click()
await page.waitForSelector('.share-empty')
await shared.reload({ waitUntil: 'networkidle' })
await shared.waitForSelector('.share-state')
if (!(await shared.locator('.share-state').innerText()).includes('失效')) throw new Error('revoked share should show invalid state')

console.log(JSON.stringify({ refsText, refLinks, link, errors }))
await browser.close()
if (errors.length) process.exitCode = 1

// 5. 长对话必须能滚动到最底部（body 在桌面端是 overflow:hidden）
{
  const longMessages = [
    ...messages,
    ...Array.from({ length: 8 }, (_, i) => ({
      id: 100 + i,
      role: 'assistant',
      content: `这是第 ${i + 1} 段很长的回答。`.repeat(30),
      reasoning: '',
      status: 'complete',
      model_name: 'DeepSeek-V4',
      input_tokens: 1,
      output_tokens: 1,
      duration_ms: 1000,
      feedback: '',
      sources: [],
      attachments: [],
      created_at: '2026-11-20T10:00:02+08:00',
    })),
  ]
  const scrollBrowser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
  const scrollContext = await scrollBrowser.newContext({ viewport: { width: 1280, height: 800 }, locale: 'zh-CN' })
  await scrollContext.route('**/api/v1/**', async (route) => {
    const p = new URL(route.request().url()).pathname
    if (p.includes('/c/public/shares/')) {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { title: '很长的分享', owner: '分享者', shared_at: '2026-11-20T10:05:00+08:00', messages: longMessages } }),
      })
    }
    return route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ code: 1002, msg: 'login required', data: null }) })
  })
  const scrollPage = await scrollContext.newPage()
  await scrollPage.goto(`${base}/s/long`, { waitUntil: 'networkidle' })
  await scrollPage.waitForSelector('.share-title-block')
  const metrics = await scrollPage.evaluate(() => {
    const el = document.querySelector('.share-page')
    if (!el) return null
    return { scrollHeight: el.scrollHeight, clientHeight: el.clientHeight, canScroll: el.scrollHeight > el.clientHeight }
  })
  if (!metrics || !metrics.canScroll) throw new Error(`share page should be scrollable: ${JSON.stringify(metrics)}`)

  await scrollPage.evaluate(() => {
    const el = document.querySelector('.share-page')
    if (el) el.scrollTop = el.scrollHeight
  })
  await scrollPage.waitForTimeout(300)
  const footVisible = await scrollPage.evaluate(() => {
    const el = document.querySelector('.share-page')
    const foot = document.querySelector('.share-foot')
    if (!el || !foot) return false
    const box = foot.getBoundingClientRect()
    return box.bottom <= window.innerHeight + 2 && box.top >= 0
  })
  if (!footVisible) throw new Error('share footer should be reachable by scrolling')
  console.log(JSON.stringify({ scroll: metrics, footVisible }))
  await scrollContext.close()
  await scrollBrowser.close()
}
