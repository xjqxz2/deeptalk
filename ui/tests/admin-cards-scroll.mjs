// 厂商卡片：铺满可用宽度，且厂商多于一行时可以横向滚动（箭头与滚轮都能到最后一个）
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const base = process.env.ADMIN_BASE || 'http://127.0.0.1:4174'
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport: { width: 1560, height: 900 }, locale: 'zh-CN' })

const admin = { id:1,email:'admin@example.com',nickname:'Admin',role:'admin',plan:'unlimited',quota_tokens:0,used_tokens:0,used_percent:0,disabled:false,created_at:'2026-01-01T00:00:00+08:00',last_login_at:null,last_login_city:'上海',last_login_ip:'127.0.0.1',avatar_text:'A',conversation_count:0,message_count:0,quota_reset_days:30 }
const makeProvider = (id, name) => ({
  id, name, code: name.toLowerCase(), kind: 'custom', base_url: `https://${name.toLowerCase()}.example/v1`,
  api_key_masked: 'sk-••••1234', has_api_key: true, timeout_sec: 60, max_concurrency: 100,
  enabled: true, verified: true, status: 'active', is_builtin: false, is_default: false,
  model_count: 1, last_latency_ms: 56, last_error: '', headers: {}, models: [],
})
const providers = ['BeeAPI','AutoDL','Maikuai','DeepSeek','Moonshot','Qwen','GLM'].map((n, i) => makeProvider(i + 1, n))

await ctx.route('**/api/v1/**', async (route) => {
  const p = new URL(route.request().url()).pathname
  const ok = (data) => route.fulfill({ status:200, contentType:'application/json', body: JSON.stringify({ code:0, msg:'ok', data }) })
  if (p.endsWith('/auth/me')) return ok({ user: admin })
  if (p.endsWith('/b/providers')) return ok({
    items: providers,
    counts: { all: providers.length, builtin: 0, custom: providers.length },
    health: { items: [], first_byte_p95_ms: 0, errors_1h: 0, available: providers.length, total: providers.length },
  })
  if (p.endsWith('/b/models')) return ok({ items: [], counts: { all: 0, enabled: 0 }, routing: { default_model_id: 0, fallback_model_id: 0 } })
  if (p.endsWith('/b/routing')) return ok({ default_model_id: 0, fallback_model_id: 0 })
  if (p.endsWith('/b/health')) return ok({ items: [], first_byte_p95_ms: 0, errors_1h: 0, available: providers.length, total: providers.length })
  return ok({ items: [], counts: { all: 0, enabled: 0, preset: 0, custom: 0 } })
})

const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(e.message))
await page.goto(`${base}/admin/providers`, { waitUntil: 'networkidle' })
await page.waitForSelector('.provider-card')

const layout = await page.evaluate(() => {
  const row = document.querySelector('.provider-row')
  if (!row) return null
  const box = row.getBoundingClientRect()
  return {
    rowWidth: Math.round(box.width),
    containerWidth: Math.round(row.parentElement.getBoundingClientRect().width),
    scrollWidth: row.scrollWidth,
    clientWidth: row.clientWidth,
    overflowX: getComputedStyle(row).overflowX,
    cards: document.querySelectorAll('.provider-card').length,
  }
})
if (!layout) throw new Error('no provider row')
// 1) 卡片行应铺满容器宽度
const gap = layout.containerWidth - layout.rowWidth
if (gap > 8) throw new Error(`row does not fill container, gap=${gap}px ${JSON.stringify(layout)}`)
// 2) 7 个厂商必须能横向滚动
if (layout.scrollWidth <= layout.clientWidth + 4) throw new Error(`row is not scrollable: ${JSON.stringify(layout)}`)

// 3) 箭头必须真的滚动
const arrows = page.locator('.card-arrows button')
const before = await page.evaluate(() => document.querySelector('.provider-row').scrollLeft)
await arrows.nth(1).click()
await page.waitForTimeout(500)
const after = await page.evaluate(() => document.querySelector('.provider-row').scrollLeft)
if (after <= before) throw new Error(`arrow did not scroll: ${before} -> ${after}`)

// 4) 一直点到最后一个厂商可见
for (let i = 0; i < 8; i += 1) {
  await arrows.nth(1).click()
  await page.waitForTimeout(220)
}
const lastVisible = await page.evaluate(() => {
  const row = document.querySelector('.provider-row')
  const cards = [...document.querySelectorAll('.provider-card')]
  const last = cards[cards.length - 1]
  const rb = row.getBoundingClientRect()
  const lb = last.getBoundingClientRect()
  return { scrollLeft: row.scrollLeft, lastRight: Math.round(lb.right), rowRight: Math.round(rb.right), reachable: lb.right <= rb.right + 2 }
})
if (!lastVisible.reachable) throw new Error(`last card not reachable: ${JSON.stringify(lastVisible)}`)

await page.screenshot({ path: '/tmp/admin-cards.png' })
console.log(JSON.stringify({ layout, before, after, lastVisible, errors }))
await browser.close()
if (errors.length) process.exitCode = 1
