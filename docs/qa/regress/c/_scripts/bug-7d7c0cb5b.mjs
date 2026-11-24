import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')
const out = {}
await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2500)

// 1) 先翻已有会话，看是否已有 A2UI 卡片
const items = page.locator('.history-item')
const n = await items.count()
out.scanned = []
for (let i = 0; i < n; i++) {
  const title = (await items.nth(i).locator('.item-title').innerText().catch(()=>'')).trim()
  await items.nth(i).click()
  await page.waitForTimeout(2500)
  const surfaces = await page.locator('a2ui-surface').count()
  out.scanned.push({ title, surfaces })
  if (surfaces > 0) break
}
console.log('SCAN:', JSON.stringify(out.scanned))

let surfaceCount = await page.locator('a2ui-surface').count()
if (!surfaceCount) {
  // 2) 发送 A2UI 请求，最多 3 次（上游偶发失败）
  for (let attempt = 1; attempt <= 3 && !surfaceCount; attempt++) {
    await page.locator('button.new-chat').click()
    await page.waitForTimeout(1000)
    await page.locator('textarea[aria-label="消息输入框"]').fill('用 A2UI 渲染一个简单卡片，带一个按钮')
    await page.locator('button[aria-label="发送消息"]').click()
    console.log('attempt', attempt, 'sent')
    for (let i = 0; i < 45; i++) {
      await page.waitForTimeout(2000)
      surfaceCount = await page.locator('a2ui-surface').count()
      if (surfaceCount) break
      const failed = await page.locator('.message.assistant .error-box').count()
      if (failed) { console.log('attempt', attempt, 'failed upstream'); break }
      const gen = await page.locator('button[aria-label="插话"]').count()
      if (!gen && i > 5) break
    }
    console.log('attempt', attempt, 'surfaces:', surfaceCount)
  }
}
out.surfaceCount = surfaceCount
if (surfaceCount) {
  await page.locator('a2ui-surface').first().scrollIntoViewIfNeeded()
  await page.waitForTimeout(800)
  await shot(page, 'BUG-7D7C0CB5-A2UI卡片')
  const inner = await page.evaluate(() => {
    const s = document.querySelector('a2ui-surface')
    const seen = (root, depth) => {
      if (!root || depth > 6) return []
      const list = []
      root.querySelectorAll('button, a, [role="button"]').forEach(b => list.push({ tag: b.tagName, text: (b.textContent||'').trim().slice(0,40), cls: b.className }))
      if (root.shadowRoot) list.push(...seen(root.shadowRoot, depth+1))
      root.querySelectorAll('*').forEach(e => { if (e.shadowRoot) list.push(...seen(e.shadowRoot, depth+1)) })
      return list
    }
    return seen(s, 0)
  })
  out.buttons = inner
  console.log('BUTTONS:', JSON.stringify(inner))
  // 点击第一个按钮（Playwright 可穿透 open shadow DOM）
  const btn = page.locator('a2ui-surface button, a2ui-surface [role="button"], a2ui-surface a').first()
  if (await btn.count()) {
    await btn.click({ timeout: 8000 }).catch(async (e) => {
      console.log('click fallback:', e.message.slice(0,100))
      const box = await btn.boundingBox()
      if (box) await page.mouse.click(box.x + box.width/2, box.y + box.height/2)
    })
    await page.waitForTimeout(900)
    out.toasts = await page.evaluate(() => [...document.querySelectorAll('.ant-message-notice-content')].map(e => e.innerText.trim()))
    await shot(page, 'BUG-7D7C0CB5-点击后反馈')
    console.log('TOASTS:', JSON.stringify(out.toasts))
  } else {
    console.log('no clickable button found in a2ui surface')
  }
}
console.log('RESULT:', JSON.stringify(out))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text)).slice(-20), null, 1))
await browser.close()
