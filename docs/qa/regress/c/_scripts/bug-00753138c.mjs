import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'

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

const state = async () => page.evaluate(() => ({
  gen: !!document.querySelector('button[aria-label="插话"]'),
  users: document.querySelectorAll('.message.user').length,
  assistants: document.querySelectorAll('.message.assistant').length,
  queued: document.querySelectorAll('.message.queued').length,
  pendingText: (document.querySelector('.interject-pending')||{}).textContent?.trim() || '',
  tags: document.querySelectorAll('.interject-tag').length,
}))
async function setText(text) {
  await page.evaluate((t) => {
    const el = document.querySelector('textarea[aria-label="消息输入框"]')
    const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value').set
    setter.call(el, t)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  }, text)
}
async function scrollBottom() {
  await page.evaluate(() => { const m = document.querySelector('.chat-content, .messages, .chat-main'); if (m) m.scrollTop = m.scrollHeight; window.scrollTo(0, document.body.scrollHeight) })
}

const results = []
for (let run = 1; run <= 3; run++) {
  log(`=== RUN ${run} ===`)
  await page.locator('button.new-chat').click()
  await page.waitForTimeout(1000)
  const r = { run, queuedSeen: false, secondRound: false, secondText: '', secondFailed: false, seconds: null }
  await setText('详细解释一下 SSE 和 WebSocket 的区别')
  await page.waitForTimeout(120)
  await page.locator('button[aria-label="发送消息"]').click()

  let gen = false
  for (let i = 0; i < 60; i++) {          // 最多 30s 等生成开始
    await page.waitForTimeout(500)
    const s = await state()
    const failed = await page.evaluate(() => [...document.querySelectorAll('.message.assistant')].some(e => /生成回答失败/.test(e.textContent||'')))
    if (failed) { log('run', run, 'first round failed upstream'); break }
    if (s.gen) { gen = true; break }
  }
  r.generating = gen
  if (!gen) { results.push(r); log('run', run, 'no generating state'); continue }

  await setText('补充一句')
  await page.waitForTimeout(120)
  await page.locator('button[aria-label="插话"]').click()
  for (let i = 0; i < 15; i++) {
    await page.waitForTimeout(400)
    const s = await state()
    if (s.queued > 0 || s.pendingText) { r.queuedSeen = true; r.pendingText = s.pendingText; break }
  }
  await scrollBottom()
  await shot(page, `BUG-00753138-run${run}-排队标记`)

  const before = await state()
  const tStart = Date.now()
  for (let i = 0; i < 90; i++) {
    await page.waitForTimeout(1000)
    const s = await state()
    if (s.assistants > before.assistants) {
      r.secondRound = true
      r.seconds = ((Date.now() - tStart) / 1000).toFixed(0)
      break
    }
  }
  // 等生成结束
  for (let i = 0; i < 90; i++) {
    await page.waitForTimeout(1000)
    const s = await state()
    if (!s.gen && s.assistants > before.assistants) break
  }
  r.answers = await page.evaluate(() => [...document.querySelectorAll('.message.assistant')].map(e => {
    const t = (e.textContent||'').replace(/\s+/g,' ').trim()
    return { len: t.length, fail: /生成回答失败/.test(t), text: t.slice(0, 100) }
  }))
  r.secondFailed = r.answers.length > 1 ? r.answers[r.answers.length-1].fail : null
  r.secondText = r.answers.length > 1 ? r.answers[r.answers.length-1].text : ''
  await scrollBottom()
  await page.waitForTimeout(600)
  await shot(page, `BUG-00753138-run${run}-第二轮回答`)
  results.push(r)
  log('run', run, 'result:', JSON.stringify({q:r.queuedSeen, second:r.secondRound, sec:r.seconds, fail:r.secondFailed, text:r.secondText.slice(0,60)}))
}
console.log('ALL RESULTS:', JSON.stringify(results, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text)).slice(-20)))
await browser.close()
