// 后台：厂商卡片填满一行 + 厂商配置里可添加多个自定义请求头
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const base = process.env.ADMIN_BASE || 'http://127.0.0.1:4174'
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport: { width: 1560, height: 900 }, locale: 'zh-CN' })

const admin = { id:1, email:'admin@example.com', nickname:'Admin', role:'admin', plan:'unlimited', quota_tokens:0, used_tokens:0, used_percent:0, disabled:false, created_at:'2026-01-01T00:00:00+08:00', last_login_at:null, last_login_city:'上海', last_login_ip:'127.0.0.1', avatar_text:'A', conversation_count:0, message_count:0, quota_reset_days:30 }
const provider = (id, name, kind, enabled, extra = {}) => ({
  id, name, code: name.toLowerCase(), kind, base_url: `https://${name.toLowerCase()}.example/v1`,
  api_key_masked: 'sk-••••1234', has_api_key: true, timeout_sec: 60, max_concurrency: 100,
  enabled, verified: true, status: enabled ? 'active' : 'disabled', is_builtin: false, is_default: false,
  model_count: 1, last_latency_ms: 56, last_error: '', headers: {}, models: [], ...extra,
})
const providers = [
  provider(1, 'BeeAPI', 'custom', false),
  provider(2, 'AutoDL', 'custom', true, { is_default: true }),
  provider(3, 'Maikuai', 'custom', true, { verified: false, headers: { 'HTTP-Referer': 'https://deeptalk.peon.top', 'X-Title': 'DeepTalk' } }),
]

await ctx.route('**/api/v1/**', async (route) => {
  const url = new URL(route.request().url())
  const p = url.pathname
  const ok = (data) => route.fulfill({ status:200, contentType:'application/json', body: JSON.stringify({ code:0, msg:'ok', data }) })
  if (p.endsWith('/b/auth/me') || p.endsWith('/c/auth/me')) return ok({ user: admin })
  if (p.endsWith('/b/providers')) return ok({
    items: providers,
    counts: { all: 3, builtin: 0, custom: 3 },
    health: { items: [], first_byte_p95_ms: 0, errors_1h: 0, available: 3, total: 3 },
  })
  if (p.endsWith('/b/models')) return ok({ items: [], counts: { all: 0, enabled: 0 }, routing: { default_model_id: 0, fallback_model_id: 0 } })
  if (p.endsWith('/b/routing')) return ok({ default_model_id: 0, fallback_model_id: 0 })
  if (p.endsWith('/b/health')) return ok({ items: [], first_byte_p95_ms: 0, errors_1h: 0, available: 1, total: 1 })
  // 其余后台接口统一给空列表，避免页面初始化时因为结构缺失报错
  return ok({ items: [], counts: { all: 0, enabled: 0, preset: 0, custom: 0 } })
})

const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(e.message))
page.on('console', m => { if (m.type()==='error' && !m.text().includes('404')) errors.push(m.text()) })

await page.goto(`${base}/admin/providers`, { waitUntil: 'networkidle' })
await page.waitForSelector('.provider-card')

// 1) 卡片应铺满一行：三张卡总宽接近容器宽度
const fill = await page.evaluate(() => {
  const row = document.querySelector('.provider-row')
  const cards = [...document.querySelectorAll('.provider-card')]
  if (!row) return null
  const rowBox = row.getBoundingClientRect()
  const gaps = (cards.length - 1) * 16
  const used = cards.reduce((sum, c) => sum + c.getBoundingClientRect().width, 0) + gaps
  return { rowWidth: Math.round(rowBox.width), used: Math.round(used), cards: cards.length }
})
if (!fill) throw new Error('provider row missing')
const fillRatio = fill.used / fill.rowWidth
if (fillRatio < 0.95) throw new Error(`cards not filling the row: ${JSON.stringify(fill)} ratio=${fillRatio.toFixed(2)}`)

// 2) 打开配置，检查请求头编辑器
await page.locator('.provider-card').filter({ hasText: 'Maikuai' }).click()
await page.waitForSelector('.ant-modal-content')
const rows = await page.locator('.header-row').count()
if (rows !== 2) throw new Error(`expected 2 existing headers, got ${rows}`)
await page.getByRole('button', { name: /添加请求头/ }).click()
const afterAdd = await page.locator('.header-row').count()
if (afterAdd !== 3) throw new Error(`add header failed: ${afterAdd}`)
await page.screenshot({ path: '/tmp/admin-headers.png' })
console.log(JSON.stringify({ fill, fillRatio: Number(fillRatio.toFixed(2)), rows, afterAdd, errors }))
await browser.close()
if (errors.length) process.exitCode = 1
