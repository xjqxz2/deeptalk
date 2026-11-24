import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')
await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2000)

const answers = async () => page.evaluate(() => [...document.querySelectorAll('.message.assistant')].map(e => {
  const t = (e.textContent||'').replace(/\s+/g,' ').trim()
  return { len: t.length, fail: /生成回答失败/.test(t), text: t.slice(0,80) }
}))
const out = []
for (let run = 1; run <= 2; run++) {
  await page.locator('button.new-chat').click(); await page.waitForTimeout(900)
  await page.locator('textarea[aria-label="消息输入框"]').fill('详细解释一下 SSE 和 WebSocket 的区别')
  await page.locator('button[aria-label="发送消息"]').click()
  // 等第一轮结束
  for (let i = 0; i < 120; i++) {
    await page.waitForTimeout(1000)
    const gen = await page.locator('button[aria-label="插话"]').count()
    const a = await answers()
    if (!gen && a.length >= 1) break
  }
  const first = (await answers())[0]
  // 正常追问（不插话）
  await page.locator('textarea[aria-label="消息输入框"]').fill('补充一句')
  await page.locator('button[aria-label="发送消息"]').click()
  for (let i = 0; i < 120; i++) {
    await page.waitForTimeout(1000)
    const gen = await page.locator('button[aria-label="插话"]').count()
    const a = await answers()
    if (!gen && a.length >= 2) break
  }
  const a = await answers()
  out.push({ run, first: first && {len:first.len, fail:first.fail}, second: a[1] && {len:a[1].len, fail:a[1].fail, text:a[1].text} })
  await page.evaluate(() => { const m=document.querySelector('.chat-content,.messages,.chat-main'); if(m) m.scrollTop=m.scrollHeight })
  await shot(page, `_control-normal-followup-run${run}`)
}
console.log('CONTROL:', JSON.stringify(out, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text))))
await browser.close()
