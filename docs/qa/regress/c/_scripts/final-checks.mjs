import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()

// ---- A. uni 深色抽屉 ----
const dctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'dark' })
const dp = await dctx.newPage()
await dp.emulateMedia({ colorScheme: 'dark' })
await dp.goto('http://127.0.0.1:5331/', { waitUntil: 'domcontentloaded' })
await dp.waitForTimeout(2500)
await dp.locator('input[type="email"]').fill(ACCOUNT.email)
await dp.locator('input[type="password"]').fill(ACCOUNT.password)
await dp.locator('button.submit-btn').click()
await dp.waitForTimeout(6500)
await dp.locator('button.menu-btn').click()
await dp.waitForTimeout(1200)
const drawer = await dp.evaluate(() => {
  const cs = (sel) => { const e = document.querySelector(sel); if (!e) return null; const s = getComputedStyle(e); return { color: s.color, bg: s.backgroundColor } }
  return { sidebar: cs('.chat-sidebar'), searchField: cs('.search-field'), usageCard: cs('.usage-card'), usageTitle: cs('.usage-title'), item: cs('.history-item') }
})
await shot(dp, 'BUG-9698E1ED-深色抽屉')
console.log('DARK DRAWER:', JSON.stringify(drawer))
await dctx.close()

// ---- B. PWA 顶栏头像菜单退出登录是否有二次确认 ----
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2500)
await page.locator('button.avatar').click()
await page.waitForTimeout(800)
const menuText = await page.locator('.ant-dropdown:not(.ant-dropdown-hidden)').innerText().catch(()=> '')
console.log('TOPBAR MENU:', JSON.stringify(menuText))
await page.locator('.ant-dropdown:not(.ant-dropdown-hidden)').getByText('退出登录', { exact: true }).click()
await page.waitForTimeout(2500)
const topbar = {
  modalCount: await page.locator('.ant-modal-confirm').count(),
  url: page.url(),
  onLogin: page.url().includes('/login'),
}
await shot(page, '_pwa-topbar-logout')
console.log('TOPBAR LOGOUT:', JSON.stringify(topbar))
await browser.close()
