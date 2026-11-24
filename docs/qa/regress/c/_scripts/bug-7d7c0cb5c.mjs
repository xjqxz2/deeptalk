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
await page.waitForTimeout(2500)

const prompts = [
  '用 A2UI 渲染一个带按钮的卡片，按钮的 action 用 event 形式（{event:{name:"confirm"}}）',
  '用 A2UI 生成一个简单的问候卡片，里面有一个「确定」按钮，点击按钮发送一个 event',
  '用 A2UI 渲染一张卡片，标题「测试」，正文「你好」，底部有一个按钮「点我」',
]
let found = null
for (let p = 0; p < prompts.length; p++) {
  await page.locator('button.new-chat').click(); await page.waitForTimeout(900)
  await page.locator('textarea[aria-label="消息输入框"]').fill(prompts[p])
  await page.locator('button[aria-label="发送消息"]').click()
  console.log('prompt', p, 'sent')
  let ok = false
  for (let i = 0; i < 40; i++) {
    await page.waitForTimeout(2000)
    const s = await page.evaluate(() => {
      const surf = document.querySelector('a2ui-surface')
      const err = [...document.querySelectorAll('*')].filter(e=>e.children.length===0 && /界面暂时无法渲染/.test(e.textContent||'')).map(e=>e.textContent.trim().slice(0,80))
      return { surf: !!surf, err, gen: !!document.querySelector('button[aria-label="插话"]') }
    })
    if (s.err.length) { console.log('render error:', JSON.stringify(s.err)); break }
    if (s.surf) { ok = true; break }
    if (!s.gen && i > 4) break
  }
  if (ok) { found = p; break }
}
console.log('found at prompt', found)
if (found !== null) {
  await page.locator('a2ui-surface').first().scrollIntoViewIfNeeded()
  await page.waitForTimeout(1000)
  await shot(page, 'BUG-7D7C0CB5-A2UI卡片')
  const buttons = await page.evaluate(() => {
    const out = []
    const walk = (root, depth) => {
      if (!root || depth > 8) return
      root.querySelectorAll('button, a, [role="button"]').forEach(b => out.push({ tag: b.tagName, text: (b.textContent||'').trim().slice(0,40), cls: b.className }))
      root.querySelectorAll('*').forEach(e => { if (e.shadowRoot) walk(e.shadowRoot, depth+1) })
    }
    const s = document.querySelector('a2ui-surface'); walk(s, 0)
    return out
  })
  console.log('BUTTONS:', JSON.stringify(buttons))
  if (buttons.length) {
    const btn = page.locator('a2ui-surface button').first()
    await btn.click({ timeout: 10000 }).catch(async (e) => {
      console.log('click err', e.message.slice(0,90))
      const box = await btn.boundingBox(); if (box) await page.mouse.click(box.x+box.width/2, box.y+box.height/2)
    })
    await page.waitForTimeout(900)
    const toasts = await page.evaluate(() => [...document.querySelectorAll('.ant-message-notice-content')].map(e => e.innerText.trim()))
    console.log('TOASTS:', JSON.stringify(toasts))
    await shot(page, 'BUG-7D7C0CB5-点击后反馈')
  }
}
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text)).slice(-10)))
await browser.close()
