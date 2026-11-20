/** 抓取小程序端（H5 调试构建）页面截图，用于和 PWA 手机端逐页比对。 */
import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('../../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const out = process.env.OUT_DIR || '/tmp/uni-shots'
import { mkdirSync } from 'node:fs'
mkdirSync(out, { recursive: true })

const user = { id:1, email:'u@example.com', nickname:'Una', role:'user', plan:'unlimited', quota_tokens:10000000, used_tokens:421000, used_percent:3, disabled:false, avatar_text:'U' }
const conv = { id:7, public_id:'u7', title:'小红书怎么挂商品链接', model_id:1, updated_at:'2026-11-20T10:00:00+08:00', message_count:2 }
const conversations = [conv,
  { id:6, public_id:'u6', title:'Go 的 SSE 代理怎么写', model_id:1, updated_at:'2026-11-20T09:00:00+08:00', message_count:2 },
  { id:5, public_id:'u5', title:'HI', model_id:1, updated_at:'2026-11-20T08:00:00+08:00', message_count:2 }]
const sources = [
  { title:'小红书个人号开通专业号流程', url:'https://example.com/a', snippet:'流程说明', viewed:true },
  { title:'商品链接挂载规则', url:'https://example.com/b', snippet:'规则', viewed:true },
]
const messages = [
  { id:1, role:'user', content:'小红书个人号怎么挂商品链接？', reasoning:'', status:'complete', model_name:'', input_tokens:12, output_tokens:0, duration_ms:0, feedback:'', sources:[], attachments:[], created_at:'2026-11-20T10:00:00+08:00' },
  { id:2, role:'assistant', content:'个人号需要先开通**专业号**并缴纳保证金，才能在笔记里挂商品链接。\n\n- 先开通专业号，完成实名与保证金缴纳\n- 在创作中心绑定店铺，商品才会出现在选品库\n- 笔记发布时选择「添加商品」，一条笔记最多挂 3 个\n\n```bash\n# 查看当前账号是否已开通\nopen https://creator.xiaohongshu.com\n```\n\n具体门槛以官方最新规则为准[1]。', reasoning:'用户问的是小红书个人号挂商品链接，这属于平台规则，需要确认最新要求。\n\n注意：个人号不能直接挂链接，必须先开通店铺或专业号。', status:'complete', model_name:'DeepSeek-V4.1-Flash', input_tokens:120, output_tokens:328, duration_ms:12400, feedback:'', sources, attachments:[], created_at:'2026-11-20T10:00:01+08:00' },
]

const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const ctx = await browser.newContext({ viewport:{width:390,height:844}, deviceScaleFactor:2, isMobile:true, hasTouch:true, locale:'zh-CN' })
await ctx.route('**/api/v1/**', async r => {
  const p = new URL(r.request().url()).pathname
  const ok = d => r.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,msg:'ok',data:d})})
  if (p.endsWith('/c/auth/me')) return ok({user})
  if (p.endsWith('/c/auth/login')) return ok({user, token:'t'})
  if (p.endsWith('/c/auth/config')) return ok({allow_registration:true,has_users:true,site_name:'DeepTalk',web_search_enabled:true})
  if (p.endsWith('/c/usage')) return ok({used_tokens:421000,quota_tokens:10000000,used_percent:3,reset_days:10,plan:'unlimited'})
  if (p.endsWith('/c/models')) return ok({items:[{id:1,model_id:'deepseek-flash',display_name:'DeepSeek-V4.1-Flash',description:'',provider_name:'DeepSeek',context_window:1000000,is_default:true}]})
  if (p.endsWith('/c/conversations/u7')) return ok({conversation:conv,messages})
  if (p.includes('/c/conversations')) return ok({items:conversations})
  return ok(null)
})
const page = await ctx.newPage()
// 1. 登录页
await page.goto('http://127.0.0.1:5273/', { waitUntil:'networkidle' })
await page.waitForSelector('input')
await page.screenshot({ path: `${out}/01-login.png` })
// 2. 登录 -> 对话页
await page.locator('input').nth(0).fill('u@example.com')
await page.locator('input').nth(1).fill('pass-1234')
await page.getByText('登录', { exact: true }).click()
await page.waitForSelector('.msg')
await page.waitForTimeout(400)
await page.screenshot({ path: `${out}/02-chat.png` })
// 3. 思维链展开
const head = page.locator('.think-head').first()
if (await head.count()) {
  await head.click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${out}/03-thinking.png` })
}
// 4. 抽屉
await page.locator('.topbar-menu').click()
await page.waitForSelector('.drawer')
await page.waitForTimeout(300)
await page.screenshot({ path: `${out}/04-drawer.png` })
console.log('uni shots done')
await browser.close()
