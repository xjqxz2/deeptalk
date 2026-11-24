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
// 新建对话
await page.locator('button.new-chat').click()
await page.waitForTimeout(1200)
// 打印模型选择器当前文本
const modelBtn = await page.evaluate(() => [...document.querySelectorAll('button')].filter(b => /Deepseek|模型/i.test(b.textContent||'')).map(b => ({cls:b.className, t:(b.textContent||'').trim()})))
console.log('MODEL CTL:', JSON.stringify(modelBtn))
await page.locator('textarea[aria-label="消息输入框"]').fill('只回复两个字：收到')
await page.locator('button[aria-label="发送消息"]').click()
for (let i = 0; i < 20; i++) {
  await page.waitForTimeout(2000)
  const state = await page.evaluate(() => ({
    gen: !!document.querySelector('button[aria-label="停止生成"],button[aria-label="插话"]'),
    err: [...document.querySelectorAll('*')].filter(e => e.children.length===0 && /生成回答失败|响应较慢/.test(e.textContent||'')).map(e=>e.textContent.trim().slice(0,60)).slice(0,3),
    last: (document.body.innerText||'').slice(-400),
  }))
  console.log(i, JSON.stringify(state).slice(0,400))
  if (!state.gen && state.err.length) break
  if (!state.gen && i>3) break
}
await shot(page, '_probe-model')
console.log('CONSOLE:', JSON.stringify(con.events.slice(-15)))
await browser.close()
