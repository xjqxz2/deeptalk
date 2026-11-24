import { launch, trackConsole, shot, sleep, ACCOUNT } from './lib.mjs'

const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')
const t0 = Date.now()
const log = (...a) => console.log(`[${((Date.now()-t0)/1000).toFixed(1)}s]`, ...a)

await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2000)
await page.locator('button.new-chat').click()
await page.waitForTimeout(1200)

const state = async () => page.evaluate(() => ({
  gen: !!document.querySelector('button[aria-label="插话"]'),
  stop: !!document.querySelector('.stop-btn'),
  users: document.querySelectorAll('.message.user').length,
  assistants: document.querySelectorAll('.message.assistant').length,
  queued: document.querySelectorAll('.message.queued').length,
  interjectTags: document.querySelectorAll('.interject-tag').length,
  pendingText: (document.querySelector('.interject-pending')||{}).textContent?.trim() || '',
  errText: [...document.querySelectorAll('.message.error, .err-text, .msg-error')].map(e=>e.textContent.trim().slice(0,80)),
  bodyTail: (document.body.innerText||'').slice(-260).replace(/\n+/g,' | '),
}))

// 1) 发长回答请求
await page.locator('textarea[aria-label="消息输入框"]').fill('详细解释一下 SSE 和 WebSocket 的区别')
await page.locator('button[aria-label="发送消息"]').click()
log('sent Q1')

// 等进入生成态
let entered = false
for (let i = 0; i < 30; i++) {
  await page.waitForTimeout(500)
  const s = await state()
  if (s.gen) { entered = true; log('generating started', JSON.stringify(s).slice(0,200)); break }
  if (s.errText.length) { log('error before gen', JSON.stringify(s.errText)); break }
}
if (!entered) { log('NEVER entered generating state'); await shot(page, '_interject-no-gen'); }

// 2) 生成中插话
await page.locator('textarea[aria-label="消息输入框"]').fill('补充一句')
log('typed interjection; send btn label =', await page.locator('.send-btn').getAttribute('aria-label'))
await page.locator('button[aria-label="插话"]').click()
log('clicked interject')

// 3) 观察排队标记
let sawQueued = false
for (let i = 0; i < 12; i++) {
  await page.waitForTimeout(500)
  const s = await state()
  if (s.queued > 0 || s.pendingText) { sawQueued = true; log('QUEUED MARK SEEN:', s.pendingText, JSON.stringify({queued:s.queued,tags:s.interjectTags})); break }
}
await shot(page, 'BUG-00753138-插话排队标记')

// 4) 等本轮结束 + 第二轮回答，最多 90s
const before = await state()
log('before wait:', JSON.stringify(before).slice(0,300))
let secondRound = false, timeline = []
for (let i = 0; i < 90; i++) {
  await page.waitForTimeout(1000)
  const s = await state()
  if (i % 5 === 0) timeline.push({ t: i, gen: s.gen, a: s.assistants, q: s.queued, tags: s.interjectTags, tail: s.bodyTail.slice(-90) })
  if (s.assistants > before.assistants) { secondRound = true; log('SECOND ROUND assistant count', before.assistants, '->', s.assistants, 'at', i, 's'); break }
  if (s.errText.length) { log('ERROR SHOWN:', JSON.stringify(s.errText)); break }
  if (!s.gen && s.assistants > before.assistants) { secondRound = true; break }
}
const final = await state()
log('FINAL:', JSON.stringify(final).slice(0,500))
await page.waitForTimeout(1500)
await shot(page, 'BUG-00753138-第二轮回答')
console.log('RESULT:', JSON.stringify({ entered, sawQueued, secondRound, before, final, timeline }, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.slice(-25), null, 1))
await browser.close()
