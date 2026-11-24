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
await page.locator('button.new-chat').click()
await page.waitForTimeout(1200)

const state = async () => page.evaluate(() => ({
  gen: !!document.querySelector('button[aria-label="插话"]'),
  users: document.querySelectorAll('.message.user').length,
  assistants: document.querySelectorAll('.message.assistant').length,
  queued: document.querySelectorAll('.message.queued').length,
  pendingText: (document.querySelector('.interject-pending')||{}).textContent?.trim() || '',
  failed: [...document.querySelectorAll('.message.assistant')].some(e => /生成回答失败/.test(e.textContent||'')),
  tail: (document.body.innerText||'').slice(-200).replace(/\n+/g,' | '),
}))

/** JS 直接写入 textarea（Vue v-model 原生 setter + input 事件）。 */
async function setText(text) {
  await page.evaluate((t) => {
    const el = document.querySelector('textarea[aria-label="消息输入框"]')
    const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value').set
    setter.call(el, t)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  }, text)
}

let attempt = 0, entered = false, res = {}
while (attempt < 3 && !entered) {
  attempt++
  log('attempt', attempt, 'send Q1')
  await setText('详细解释一下 SSE 和 WebSocket 的区别')
  await page.waitForTimeout(150)
  const sendBtn = page.locator('button[aria-label="发送消息"]')
  if (await sendBtn.count()) await sendBtn.click()
  else { log('no send btn, current state', JSON.stringify(await state()).slice(0,200)); }

  for (let i = 0; i < 40; i++) {   // 最多 20s
    await page.waitForTimeout(500)
    const s = await state()
    if (s.failed) { log('attempt failed (upstream):', s.tail.slice(-120)); break }
    if (s.gen) { entered = true; log('GENERATING at attempt', attempt, JSON.stringify(s).slice(0,160)); break }
  }
  if (!entered) { await page.waitForTimeout(500) }
}

res.entered = entered
res.attempt = attempt
if (entered) {
  // 生成中立刻插话
  await setText('补充一句')
  await page.waitForTimeout(120)
  const btn = page.locator('button[aria-label="插话"]')
  log('clicking interject, label=', await page.locator('.send-btn').getAttribute('aria-label'))
  await btn.click()
  log('clicked interject')

  let sawQueued = false
  for (let i = 0; i < 14; i++) {
    await page.waitForTimeout(400)
    const s = await state()
    if (s.queued > 0 || s.pendingText) { sawQueued = true; log('QUEUED SEEN:', s.pendingText); break }
  }
  res.sawQueued = sawQueued
  await shot(page, 'BUG-00753138-插话排队标记')

  const before = await state()
  log('waiting for second round; before=', JSON.stringify(before).slice(0,200))
  let second = false, timeline = []
  for (let i = 0; i < 90; i++) {
    await page.waitForTimeout(1000)
    const s = await state()
    if (i % 6 === 0) timeline.push({ t: i, gen: s.gen, a: s.assistants, q: s.queued, tags: s.users })
    if (s.assistants > before.assistants) { second = true; log('SECOND ROUND at', i, 's:', before.assistants, '->', s.assistants); break }
  }
  res.secondRound = second
  res.timeline = timeline
  // 等第二轮收尾（生成结束或超时），抓最终回答文本
  let done = false
  for (let i = 0; i < 90; i++) {
    await page.waitForTimeout(1000)
    const s = await state()
    if (!s.gen) { done = true; log('generation finished at +', i, 's'); break }
  }
  res.secondDone = done
  const fin = await state()
  res.final = fin
  res.answers = await page.evaluate(() => [...document.querySelectorAll('.message.assistant')].map(e => {
    const t = (e.textContent||'').replace(/\s+/g,' ').trim()
    return { len: t.length, fail: /生成回答失败/.test(t), text: t.slice(0, 160) }
  }))
  log('final:', JSON.stringify(fin).slice(0,240))
  await page.waitForTimeout(1200)
  await shot(page, 'BUG-00753138-插话后第二轮回答')
}
console.log('RESULT:', JSON.stringify(res, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text)).slice(-25), null, 1))
await browser.close()
