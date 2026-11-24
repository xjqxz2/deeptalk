import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const UNI = 'http://127.0.0.1:5331/'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'light' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
const out = {}
await page.goto(UNI, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2500)
await page.locator('input[type="email"]').fill(ACCOUNT.email)
await page.locator('input[type="password"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForTimeout(6500)

out.modelBefore = (await page.locator('.model-trigger').innerText()).trim()
await page.locator('.model-trigger').click()
await page.waitForTimeout(1200)
out.sheet = await page.evaluate(() => {
  const nodes = [...document.querySelectorAll('[class*="actionsheet"], [class*="action-sheet"]')]
  return nodes.map(n => ({ cls: n.className, text: (n.innerText||'').slice(0,200) }))
})
out.cells = await page.locator('.uni-actionsheet__cell').allInnerTexts().catch(()=>[])
await shot(page, 'BUG-96060A5E-模型选择器')
console.log('SHEET:', JSON.stringify(out.sheet).slice(0,400))
console.log('CELLS:', JSON.stringify(out.cells))
if (out.cells.length > 1) {
  await page.locator('.uni-actionsheet__cell').nth(1).click()
  await page.waitForTimeout(2500)
}
out.modelAfter = (await page.locator('.model-trigger').innerText()).trim()
await shot(page, 'BUG-96060A5E-切换后模型')
console.log('MODEL:', JSON.stringify(out.modelBefore), '->', JSON.stringify(out.modelAfter))

// 赞 / 踩（每条助手消息 2 个 icon：赞、踩）
await page.locator('.messages-scroll').evaluate(el => { el.scrollTop = el.scrollHeight })
await page.waitForTimeout(800)
const lastMsg = page.locator('.message.assistant').last()
const icons = lastMsg.locator('.message-actions .action.icon')
out.iconCount = await icons.count()
await icons.nth(0).click({ force: true })
await page.waitForTimeout(1800)
out.likeClass = await icons.nth(0).getAttribute('class')
out.likeActive = await page.locator('.message-actions .action.icon.active').count()
await shot(page, 'BUG-96060A5E-点赞高亮')
await icons.nth(1).click({ force: true })
await page.waitForTimeout(1800)
out.afterDislike = { like: await icons.nth(0).getAttribute('class'), dislike: await icons.nth(1).getAttribute('class'), activeTotal: await page.locator('.message-actions .action.icon.active').count() }
await shot(page, 'BUG-96060A5E-点踩高亮')
out.pageErrors = con.events.filter(e => e.kind === 'pageerror')
out.onRateErr = con.events.filter(e => /onRate/.test(e.text))
console.log('RESULT:', JSON.stringify(out, null, 1))
await browser.close()
