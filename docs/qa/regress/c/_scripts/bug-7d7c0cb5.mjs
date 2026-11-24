import { launch, trackConsole, shot, sleep, ACCOUNT } from './lib.mjs'

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

// 顺带核对顶栏头像菜单是否会弹确认框（只观察，不真正退出）
await page.locator('button.avatar').click()
await page.waitForTimeout(800)
const topMenu = await page.evaluate(() => [...document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden)')].map(e => e.innerText.trim()))
console.log('TOPBAR DROPDOWNS:', JSON.stringify(topMenu))
await page.keyboard.press('Escape')
await page.waitForTimeout(400)

// 新建对话，发一条 A2UI 请求
const ta = page.locator('textarea[aria-label="消息输入框"]')
await ta.fill('用 A2UI 渲染一个简单卡片，带一个按钮')
await page.locator('button[aria-label="发送消息"]').click()
console.log('sent A2UI prompt at', new Date().toISOString())

let found = false
for (let i = 0; i < 60; i++) {
  await page.waitForTimeout(2000)
  const n = await page.locator('a2ui-surface').count()
  if (n > 0) { found = true; break }
  const gen = await page.locator('button[aria-label="停止生成"], button[aria-label="插话"]').count()
  if (i % 5 === 0) console.log(`t=${i*2}s generatingBtn=${gen} surfaces=${n} msgs=${await page.locator('.message-item, [class*=message]').count()}`)
  if (!gen && i > 6) break
}
console.log('A2UI surface found:', found)
const info = { found, toast: null, buttons: [], beforeShot: null }

if (found) {
  await page.locator('a2ui-surface').first().scrollIntoViewIfNeeded()
  await page.waitForTimeout(800)
  // 卡片里的按钮可能在 shadowRoot 内，Playwright 会穿透 open shadow root
  const btns = page.locator('a2ui-surface button, a2ui-surface [role="button"], a2ui-surface a')
  const count = await btns.count()
  info.buttons = count
  console.log('a2ui buttons:', count)
  if (count > 0) {
    const label = (await btns.first().innerText().catch(() => '')).trim()
    info.firstButton = label
    await btns.first().click({ timeout: 8000 }).catch(async (e) => {
      console.log('normal click failed:', e.message.slice(0,120))
      const box = await btns.first().boundingBox()
      if (box) await page.mouse.click(box.x + box.width/2, box.y + box.height/2)
    })
    await page.waitForTimeout(700)
    // ant message toast
    const toasts = await page.evaluate(() => [...document.querySelectorAll('.ant-message-notice-content, .ant-message')].map(e => e.innerText.trim()))
    info.toast = toasts
    console.log('toasts:', JSON.stringify(toasts))
    await shot(page, 'BUG-7D7C0CB5-点击后反馈')
  }
} else {
  await shot(page, '_a2ui-未出现')
}
await shot(page, '_a2ui-card')
console.log('RESULT:', JSON.stringify(info))
console.log('CONSOLE:', JSON.stringify(con.events.slice(-40), null, 1))
await browser.close()
