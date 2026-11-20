// 侧栏标题：过长时必须显示省略号，并且悬停能看到完整标题
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const base = process.env.BASE_URL || 'http://127.0.0.1:4173'
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport:{width: 1100, height: 760}, locale:'zh-CN' })
const user = { id:1,email:'u@e.com',nickname:'U',role:'user',plan:'unlimited',quota_tokens:0,used_tokens:0,used_percent:0,disabled:false,created_at:'2026-01-01T00:00:00+08:00',last_login_at:null,last_login_city:'上海',last_login_ip:'127.0.0.1',avatar_text:'U',conversation_count:2,message_count:1,quota_reset_days:30 }
const longTitle = '想咨询一下 Zerotier 组网方案在企业内网场景下的具体配置与性能表现'
const convs = [
  { id: 1, public_id: 'c1', title: longTitle, model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 1 },
  { id: 2, public_id: 'c2', title: 'HI', model_id: 1, updated_at: '2026-11-20T09:00:00+08:00', message_count: 1 },
]
await ctx.route('**/api/v1/**', async r => {
  const p = new URL(r.request().url()).pathname
  const ok = d => r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,msg:'ok',data:d})})
  if (p.endsWith('/c/auth/me')) return ok({user})
  if (p.endsWith('/c/usage')) return ok({used_tokens:0,quota_tokens:0,used_percent:0,reset_days:30,plan:'unlimited'})
  if (p.endsWith('/c/models')) return ok({items:[{id:1,model_id:'x',display_name:'DeepSeek-V4',description:'',provider_name:'D',context_window:1,is_default:true}]})
  if (p.endsWith('/c/conversations/c2')) return ok({conversation:convs[1],messages:[]})
  if (p.endsWith('/c/conversations/c1')) return ok({conversation:convs[0],messages:[]})
  if (p.includes('/c/conversations')) return ok({items:convs})
  return ok(null)
})
const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(e.message))
await page.goto(`${base}/chat/c1`, { waitUntil:'networkidle' })
await page.waitForSelector('.history-item')
const result = await page.evaluate(() => {
  const items = [...document.querySelectorAll('.history-item .item-title')]
  return items.map((el) => ({
    text: el.textContent,
    title: el.getAttribute('title'),
    truncated: el.scrollWidth > el.clientWidth + 1,
    textOverflow: getComputedStyle(el).textOverflow,
    whiteSpace: getComputedStyle(el).whiteSpace,
    rendered: el.getBoundingClientRect().width,
  }))
})
const first = result[0]
if (!first.truncated) throw new Error(`long title should overflow: ${JSON.stringify(first)}`)
if (first.textOverflow !== 'ellipsis') throw new Error(`expected ellipsis: ${JSON.stringify(first)}`)
if (first.title !== longTitle) throw new Error('full title missing in title attribute')
await page.screenshot({ path: '/tmp/sidebar-titles.png' })
console.log(JSON.stringify({ result, errors }))
await browser.close()
if (errors.length) process.exitCode = 1
