import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const UNI = 'http://127.0.0.1:5331/'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'light' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
const reqs = []
page.on('request', r => { if (/conversations/.test(r.url()) && r.method()==='GET') reqs.push(r.url()) })
const out = {}
await page.goto(UNI, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2500)
await page.locator('input[type="email"]').fill(ACCOUNT.email)
await page.locator('input[type="password"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForTimeout(6000)

await page.locator('button.menu-btn').click()
await page.waitForTimeout(1000)
out.before = { count: await page.locator('.history-item').count(), titles: (await page.locator('.history-item .item-title').allInnerTexts()).slice(0,8) }
await shot(page, '_uni-drawer-all')

const search = page.locator('input[aria-label="搜索对话"]')
await search.click()
await search.fill('SSE')
await page.waitForTimeout(2500)
out.afterSSE = {
  inputValue: await search.inputValue(),
  count: await page.locator('.history-item').count(),
  titles: (await page.locator('.history-item .item-title').allInnerTexts()).slice(0,8),
}
await page.keyboard.press('Enter')
await page.waitForTimeout(2000)
out.afterSSEEnter = { count: await page.locator('.history-item').count(), titles: (await page.locator('.history-item .item-title').allInnerTexts()).slice(0,8) }
await shot(page, 'BUG-F305D150-搜索过滤')

await search.fill('zzzz')
await page.waitForTimeout(2500)
out.afterZzzz = {
  inputValue: await search.inputValue(),
  count: await page.locator('.history-item').count(),
  empty: await page.locator('.history-empty').innerText().catch(()=>''),
}
await page.keyboard.press('Enter')
await page.waitForTimeout(2000)
out.afterZzzzEnter = {
  count: await page.locator('.history-item').count(),
  empty: await page.locator('.history-empty').innerText().catch(()=>''),
  visibleText: (await page.locator('.chat-sidebar').innerText()).slice(0, 300),
}
await shot(page, 'BUG-F305D150-空态')
out.requests = reqs.slice(-6)
console.log('RESULT:', JSON.stringify(out, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,15)))
await browser.close()
