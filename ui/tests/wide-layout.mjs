// 超宽屏（2560）下聊天区域是否居中、正文宽度是否合理
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const base = process.env.BASE_URL || 'http://127.0.0.1:4173'
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport: { width: 2560, height: 1305 }, locale: 'zh-CN' })
const messages = [
  { id:1, role:'user', content:'聊天和写代码 token 差异', reasoning:'', status:'complete', model_name:'', input_tokens:1, output_tokens:0, duration_ms:0, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:00+08:00' },
  { id:2, role:'assistant', content:'这是一段很长的回答，用来测试超宽屏下的行宽是否合适。'.repeat(6), reasoning:'', status:'complete', model_name:'DeepSeek-V4.1-Flash', input_tokens:1, output_tokens:1, duration_ms:2000, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:01+08:00' },
]
await ctx.route('**/api/v1/**', async r => {
  const p = new URL(r.request().url()).pathname
  const ok = d => r.fulfill({ status:200, contentType:'application/json', body: JSON.stringify({ code:0, msg:'ok', data:d }) })
  if (p.endsWith('/c/auth/me')) return ok({ user:{ id:1,email:'w@e.com',nickname:'W',role:'user',plan:'unlimited',quota_tokens:0,used_tokens:0,used_percent:0,disabled:false,created_at:'2026-01-01T00:00:00+08:00',last_login_at:null,last_login_city:'上海',last_login_ip:'1.1.1.1',avatar_text:'W',conversation_count:1,message_count:2,quota_reset_days:30 } })
  if (p.endsWith('/c/usage')) return ok({ used_tokens:100,quota_tokens:1000,used_percent:3,reset_days:10,plan:'unlimited' })
  if (p.endsWith('/c/models')) return ok({ items:[{id:1,model_id:'x',display_name:'DeepSeek-V4.1-Flash',description:'',provider_name:'D',context_window:1,is_default:true}] })
  if (p.endsWith('/c/conversations/w1')) return ok({ conversation:{ id:1,public_id:'w1',title:'聊天和写代码 token 差异',model_id:1,updated_at:'2026-11-20T10:00:00+08:00',message_count:2 }, messages })
  if (p.includes('/c/conversations')) return ok({ items:[{ id:1,public_id:'w1',title:'聊天和写代码 token 差异',model_id:1,updated_at:'2026-11-20T10:00:00+08:00',message_count:2 }] })
  return ok(null)
})
const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(e.message))
await page.goto(`${base}/chat/w1`, { waitUntil: 'networkidle' })
await page.waitForSelector('.messages-column')
const m = await page.evaluate(() => {
  const pick = (sel) => {
    const el = document.querySelector(sel)
    if (!el) return null
    const r = el.getBoundingClientRect()
    return { left: Math.round(r.left), right: Math.round(r.right), width: Math.round(r.width), center: Math.round((r.left + r.right) / 2) }
  }
  const main = document.querySelector('.chat-main') || document.querySelector('main')
  const mainBox = main ? main.getBoundingClientRect() : null
  return {
    viewport: window.innerWidth,
    main: mainBox ? { left: Math.round(mainBox.left), right: Math.round(mainBox.right), width: Math.round(mainBox.width), center: Math.round((mainBox.left + mainBox.right) / 2) } : null,
    column: pick('.messages-column'),
    prose: pick('.message.assistant .prose'),
    composer: pick('.composer-box') || pick('.composer'),
  }
})
// 超宽屏：正文列要适度加宽，并且仍然水平居中
const drift = Math.abs(m.column.center - m.main.center)
if (drift > 2) throw new Error(`column off-centre by ${drift}px`)
if (m.column.width < 1200) throw new Error(`column too narrow on 2560: ${m.column.width}`)
if (m.composer && m.composer.width > 960) throw new Error(`composer should stay compact: ${m.composer.width}`)
if (m.composer && Math.abs(m.composer.center - m.main.center) > 2) throw new Error('composer off-centre')
if (m.prose && m.prose.width < 1100) throw new Error(`text column too narrow: ${m.prose.width}`)
await page.screenshot({ path: '/tmp/wide-layout.png' })
console.log(JSON.stringify({ ...m, drift, errors }))
await browser.close()
