// iOS PWA 布局检查：内容是否水平居中、有没有横向溢出
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 3, isMobile: true, hasTouch: true, locale: 'zh-CN' })
const user = { id:1,email:'m@e.com',nickname:'M',role:'user',plan:'unlimited',quota_tokens:0,used_tokens:0,used_percent:0,disabled:false,created_at:'2026-01-01T00:00:00+08:00',last_login_at:null,last_login_city:'上海',last_login_ip:'127.0.0.1',avatar_text:'M',conversation_count:1,message_count:2,quota_reset_days:30 }
const conv = { id: 5, public_id: 'm5', title: 'AI 电话营销服务调研', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const long = '上面这四块内容对应的就是前面聊到的全部结论：技术流程、价格区间、合规红线，以及自用与开公司两条路径。界面上最需要关注的是第三块和第四块：红线部分与该行业风险直接相关。'
const messages = [
  { id:1, role:'user', content:'我想了解 AI 电话营销', reasoning:'', status:'complete', model_name:'', input_tokens:1, output_tokens:0, duration_ms:0, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:00+08:00' },
  { id:2, role:'assistant', content: long, reasoning:'思考', status:'complete', model_name:'DeepSeek-V4', input_tokens:1, output_tokens:1, duration_ms:2000, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:01+08:00' },
]
await ctx.route('**/api/v1/**', async r => {
  const p = new URL(r.request().url()).pathname
  const ok = d => r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,msg:'ok',data:d})})
  if (p.endsWith('/c/auth/me')) return ok({user})
  if (p.endsWith('/c/usage')) return ok({used_tokens:0,quota_tokens:0,used_percent:0,reset_days:30,plan:'unlimited'})
  if (p.endsWith('/c/models')) return ok({items:[{id:1,model_id:'x',display_name:'DeepSeek-V4',description:'',provider_name:'D',context_window:1,is_default:true}]})
  if (p.endsWith('/c/conversations/m5')) return ok({conversation:conv, messages})
  if (p.includes('/c/conversations')) return ok({items:[conv]})
  return ok(null)
})
const page = await ctx.newPage()
await page.goto('http://127.0.0.1:4173/chat/m5', { waitUntil:'networkidle' })
await page.waitForSelector('.messages-column')
const metrics = await page.evaluate(() => {
  const doc = document.documentElement
  const box = sel => {
    const el = document.querySelector(sel)
    if (!el) return null
    const r = el.getBoundingClientRect()
    return { left: Math.round(r.left), right: Math.round(r.right), width: Math.round(r.width), center: Math.round((r.left + r.right) / 2) }
  }
  const prose = document.querySelector('.message.assistant .prose')
  const pr = prose?.getBoundingClientRect()
  const avatar = document.querySelector('.message.assistant .dt-brand-mark')
  const ar = avatar?.getBoundingClientRect()
  return {
    viewport: { width: window.innerWidth, center: window.innerWidth / 2 },
    scrollWidth: doc.scrollWidth,
    bodyScrollWidth: document.body.scrollWidth,
    overflowing: doc.scrollWidth > window.innerWidth,
    column: box('.messages-column'),
    composer: box('.composer-box'),
    message: box('.message.assistant'),
    topbar: box('.chat-topbar'),
    prose: pr ? { left: Math.round(pr.left), right: Math.round(pr.right), center: Math.round((pr.left + pr.right) / 2) } : null,
    avatar: ar ? { left: Math.round(ar.left), width: Math.round(ar.width) } : null,
  }
})
// 正文必须水平居中，偏差不超过 2px
const drift = Math.abs(metrics.prose.center - metrics.viewport.center)
if (drift > 2) throw new Error(`assistant text is off-centre by ${drift}px`)
if (metrics.overflowing) throw new Error('page has horizontal overflow')
if (metrics.composer && Math.abs(metrics.composer.center - metrics.viewport.center) > 2) throw new Error('composer is off-centre')
console.log(JSON.stringify({ drift, ...metrics }, null, 2))
await page.screenshot({ path: '/tmp/mobile-layout.png' })
await page.setViewportSize({ width: 390, height: 844 })
await page.screenshot({ path: '/tmp/mobile-layout-top.png' })
await browser.close()
