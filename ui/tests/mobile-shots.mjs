/**
 * 抓取 PWA 手机端关键页面截图，作为小程序端 1:1 还原的设计参照。
 * 输出到 /tmp/mobile-shots/*.png（390×844，2 倍图）。
 */
import path from 'node:path'
import { chromium } from 'playwright-core'

const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const out = process.env.OUT_DIR || '/tmp/mobile-shots'
const base = 'http://127.0.0.1:4173'

const user = { id:1, email:'u@example.com', nickname:'Una', role:'user', plan:'unlimited', quota_tokens:0, used_tokens:0, used_percent:0, disabled:false, created_at:'2026-01-01T00:00:00+08:00', last_login_at:null, last_login_city:'上海', last_login_ip:'127.0.0.1', avatar_text:'U', conversation_count:3, message_count:8, quota_reset_days:30 }
const conv = { id:7, public_id:'u7', title:'小红书怎么挂商品链接', model_id:1, updated_at:'2026-11-20T10:00:00+08:00', message_count:2 }
const conversations = [
  conv,
  { id:6, public_id:'u6', title:'Go 的 SSE 代理怎么写', model_id:1, updated_at:'2026-11-20T09:00:00+08:00', message_count:2 },
  { id:5, public_id:'u5', title:'HI', model_id:1, updated_at:'2026-11-20T08:00:00+08:00', message_count:2 },
]
const sources = [
  { title:'小红书个人号开通专业号流程', url:'https://example.com/a', snippet:'流程说明', viewed:true },
  { title:'商品链接挂载规则', url:'https://example.com/b', snippet:'规则', viewed:true },
]
const messages = [
  { id:1, role:'user', content:'小红书个人号怎么挂商品链接？', reasoning:'', status:'complete', model_name:'', input_tokens:12, output_tokens:0, duration_ms:0, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:00+08:00' },
  { id:2, role:'assistant', content:'个人号需要先开通**专业号**并缴纳保证金，才能在笔记里挂商品链接。\n\n- 先开通专业号，完成实名与保证金缴纳\n- 在创作中心绑定店铺，商品才会出现在选品库\n- 笔记发布时选择「添加商品」，一条笔记最多挂 3 个\n\n```bash\n# 查看当前账号是否已开通\nopen https://creator.xiaohongshu.com\n```\n\n具体门槛以官方最新规则为准[1]。', reasoning:'用户问的是小红书个人号挂商品链接，这属于平台规则，需要确认最新要求。\n\n注意：个人号不能直接挂链接，必须先开通店铺或专业号。', status:'complete', model_name:'DeepSeek-V4.1-Flash', input_tokens:120, output_tokens:328, duration_ms:12400, feedback:'', sources, attachments:[], created_at:'2026-11-20T10:00:01+08:00' },
]

const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const context = await browser.newContext({ viewport:{width:390,height:844}, deviceScaleFactor:2, isMobile:true, hasTouch:true, locale:'zh-CN', timezoneId:'Asia/Shanghai' })
let loggedIn = false
await context.route('**/api/v1/**', async (route) => {
  const p = new URL(route.request().url()).pathname
  const ok = (data) => route.fulfill({ status:200, contentType:'application/json', body: JSON.stringify({ code:0, msg:'ok', data }) })
  if (p.endsWith('/c/auth/config')) return ok({ allow_registration:true, has_users:true, site_name:'DeepTalk', web_search_enabled:true })
  if (p.endsWith('/c/auth/me')) return loggedIn ? ok({ user }) : route.fulfill({ status:401, contentType:'application/json', body: JSON.stringify({ code:1002, msg:'login required', data:null }) })
  if (!loggedIn) return route.fulfill({ status:401, contentType:'application/json', body: JSON.stringify({ code:1002, msg:'login required', data:null }) })
  if (p.endsWith('/c/usage')) return ok({ used_tokens:421000, quota_tokens:10000000, used_percent:3, reset_days:10, plan:'unlimited' })
  if (p.endsWith('/c/models')) return ok({ items:[{ id:1, model_id:'deepseek-flash', display_name:'DeepSeek-V4.1-Flash', description:'', provider_name:'DeepSeek', context_window:1000000, is_default:true }] })
  if (p.endsWith('/c/conversations/u7')) return ok({ conversation:conv, messages })
  if (p.includes('/c/conversations')) return ok({ items:conversations })
  return ok(null)
})

const page = await context.newPage()
// 1. 登录页
await page.goto(`${base}/login`, { waitUntil:'networkidle' })
await page.waitForSelector('.submit-btn')
await page.screenshot({ path: `${out}/01-login.png` })

// 2. 空态
loggedIn = true
await page.goto(`${base}/`, { waitUntil:'networkidle' })
await page.waitForSelector('.chat-empty')
await page.screenshot({ path: `${out}/02-empty.png` })

// 3. 对话页（含思维链、列表、代码、引用）
await page.goto(`${base}/chat/u7`, { waitUntil:'networkidle' })
await page.waitForSelector('.message')
await page.screenshot({ path: `${out}/03-chat.png` })

// 4. 思维链展开
const thinking = page.locator('.thinking-head').first()
if (await thinking.count()) {
  const expanded = await page.locator('.thinking').first().evaluate((el) => el.classList.contains('expanded'))
  if (!expanded) await thinking.click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${out}/04-thinking.png` })
}

// 5. 历史会话抽屉
const menu = page.locator('button[aria-label="打开侧栏"], .sidebar-toggle, .menu-btn').first()
if (await menu.count()) {
  await menu.click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: `${out}/05-drawer.png` })
}

// 6. 账号菜单（手机端顶栏右侧是头像，分享入口在侧栏里）
await page.goto(`${base}/chat/u7`, { waitUntil:'networkidle' })
await page.waitForSelector('.message')
const avatar = page.locator('.avatar').first()
if (await avatar.count()) {
  await avatar.click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: `${out}/06-account.png` })
}

console.log('shots done')
await browser.close()
