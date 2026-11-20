// 逐页烟测：确认按需注册后每个页面的 antd 组件都能正常渲染，没有控制台错误。
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport:{width:1200,height:860}, locale:'zh-CN', timezoneId:'Asia/Shanghai' })
const user = { id:1,email:'u@e.com',nickname:'Uma',role:'user',plan:'unlimited',quota_tokens:0,used_tokens:0,used_percent:0,disabled:false,created_at:'2026-01-01T00:00:00+08:00',last_login_at:null,last_login_city:'上海',last_login_ip:'127.0.0.1',avatar_text:'U',conversation_count:2,message_count:2,quota_reset_days:30 }
const conv = { id: 9, public_id: 'p9', title: '测试会话', model_id: 1, updated_at: '2026-11-20T10:00:00+08:00', message_count: 2 }
const messages = [
  { id:1, role:'user', content:'你好', reasoning:'', status:'complete', model_name:'', input_tokens:1, output_tokens:0, duration_ms:0, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:00+08:00' },
  { id:2, role:'assistant', content:'**加粗** 和 `code`\n\n```go\nfunc main() {\n\tfmt.Println("hi")\n}\n```\n\n```brainfuck\n+++\n```', reasoning:'思考一下', status:'complete', model_name:'DeepSeek-V4', input_tokens:1, output_tokens:1, duration_ms:2000, feedback:'up', sources:[{title:'来源',url:'https://example.com',snippet:'s',viewed:true}], attachments:[], created_at:'2026-11-20T10:00:01+08:00' },
]
let loggedIn = false
await ctx.route('**/api/v1/**', async r => {
  const p = new URL(r.request().url()).pathname
  const ok = d => r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,msg:'ok',data:d})})
  const deny = () => r.fulfill({status:401,contentType:'application/json',body:JSON.stringify({code:1002,msg:'login required',data:null})})
  if (p.endsWith('/c/auth/me')) return loggedIn ? ok({user}) : deny()
  if (p.endsWith('/c/auth/config')) return ok({allow_registration:true,has_users:true,site_name:'DeepTalk',web_search_enabled:true})
  if (!loggedIn) return deny()
  if (p.endsWith('/c/usage')) return ok({used_tokens:100,quota_tokens:1000,used_percent:10,reset_days:20,plan:'unlimited'})
  if (p.endsWith('/c/models')) return ok({items:[{id:1,model_id:'x',display_name:'DeepSeek-V4',description:'',provider_name:'D',context_window:1,is_default:true}]})
  if (p.endsWith('/c/conversations/p9')) return ok({conversation:conv,messages})
  if (p.includes('/c/conversations')) return ok({items:[conv]})
  return ok(null)
})
const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(e.message))
page.on('console', m => { if (m.type()==='error' && !m.text().includes('401')) errors.push(m.text()) })
const checks = []
async function visit(name, url, selector) {
  await page.goto(url, { waitUntil:'networkidle' })
  await page.waitForSelector(selector, { timeout: 15000 })
  checks.push(name)
}
await visit('login', 'http://127.0.0.1:4173/login', '.submit-btn')
await visit('register', 'http://127.0.0.1:4173/register', '.submit-btn')
loggedIn = true
await visit('empty', 'http://127.0.0.1:4173/', '.chat-empty')
await visit('session', 'http://127.0.0.1:4173/chat/p9', '.messages-column')
await visit('notfound', 'http://127.0.0.1:4173/nope', '.not-found')
// 交互：账号菜单、分享弹窗
await page.goto('http://127.0.0.1:4173/chat/p9', { waitUntil:'networkidle' })
await page.waitForSelector('.messages-column')
await page.locator('.avatar').click()
await page.waitForSelector('.ant-dropdown-menu')
await page.keyboard.press('Escape')
await page.locator('button[aria-label="分享对话"]').click()
await page.waitForSelector('.ant-modal-content')
checks.push('share-modal')
// 代码高亮仍生效，且只注册了常见语言
const hl = await page.locator('.dt-code-pre code .hljs-keyword').count()
console.log(JSON.stringify({ checks, errors, hljsKeywordSpans: hl }))
await browser.close()
if (errors.length) process.exitCode = 1
