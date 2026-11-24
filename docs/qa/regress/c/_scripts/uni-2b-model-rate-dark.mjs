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
await page.waitForTimeout(6000)

// ---- 模型选择 ----
out.modelBefore = (await page.locator('.model-trigger').innerText()).trim()
await page.locator('.model-trigger').click()
await page.waitForTimeout(1200)
out.popupVisible = await page.locator('.van-popup').isVisible().catch(()=>false)
out.popupBox = await page.locator('.van-popup').boundingBox().catch(()=>null)
out.pickerTitle = await page.locator('.van-picker__title').innerText().catch(()=>'')
out.options = await page.locator('.van-picker-column__item').allInnerTexts().catch(()=>[])
await shot(page, 'BUG-96060A5E-模型选择器')
console.log('PICKER:', JSON.stringify({visible:out.popupVisible, box:out.popupBox, title:out.pickerTitle, options:out.options.slice(0,10)}))
if (out.options.length > 1) {
  await page.locator('.van-picker-column__item').nth(1).click()
  await page.waitForTimeout(500)
  await page.locator('.van-picker__confirm').click()
  await page.waitForTimeout(2000)
}
out.modelAfter = (await page.locator('.model-trigger').innerText().catch(()=>'')).trim()
await shot(page, 'BUG-96060A5E-切换后模型')
console.log('MODEL:', JSON.stringify(out.modelBefore), '->', JSON.stringify(out.modelAfter))

// ---- 赞/踩 ----
const icons = page.locator('.message.assistant .message-actions .action.icon')
out.iconCount = await icons.count()
if (out.iconCount > 0) {
  const first = icons.first()
  await page.locator('.messages-scroll').evaluate((el) => { el.scrollTop = el.scrollHeight })
  await page.waitForTimeout(600)
  await first.click({ force: true })
  await page.waitForTimeout(1500)
  out.likeClass = await first.getAttribute('class')
  out.activeCount = await page.locator('.message-actions .action.icon.active').count()
  await shot(page, 'BUG-96060A5E-点赞高亮')
  // 再点踩
  const second = icons.nth(1)
  if (await second.count()) {
    await second.click({ force: true })
    await page.waitForTimeout(1200)
    out.dislikeClass = await second.getAttribute('class')
    out.activeAfterDislike = await page.locator('.message-actions .action.icon.active').count()
    await shot(page, '_uni-dislike')
  }
}
out.pageErrors = con.events.filter(e => e.kind === 'pageerror')
out.onRateErr = con.events.filter(e => /onRate/.test(e.text))
console.log('RATE:', JSON.stringify({count: out.iconCount, likeClass: out.likeClass, active: out.activeCount, dislikeClass: out.dislikeClass, pageErrors: out.pageErrors, onRate: out.onRateErr}, null, 1))
await ctx.close()

// ---- 深色聊天页 ----
const dctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'dark' })
const dpage = await dctx.newPage()
await dpage.emulateMedia({ colorScheme: 'dark' })
const dcon = trackConsole(dpage, 'uni-dark')
await dpage.goto(UNI, { waitUntil: 'domcontentloaded' })
await dpage.waitForTimeout(2500)
await dpage.locator('input[type="email"]').fill(ACCOUNT.email)
await dpage.locator('input[type="password"]').fill(ACCOUNT.password)
const typed = await dpage.evaluate(() => { const i = document.querySelector('input[type="email"]'); const s = getComputedStyle(i); return { color: s.color, bg: s.backgroundColor } })
await shot(dpage, 'BUG-9698E1ED-深色登录页输入文字')
await dpage.locator('button.submit-btn').click()
await dpage.waitForTimeout(6500)
await shot(dpage, 'BUG-9698E1ED-深色聊天页')
out.dark = await dpage.evaluate(() => {
  const cs = (sel) => { const e = document.querySelector(sel); if (!e) return null; const s = getComputedStyle(e); return { color: s.color, bg: s.backgroundColor } }
  const prose = document.querySelector('.message.assistant .prose')
  return {
    body: cs('body'), scroll: cs('.messages-scroll'), column: cs('.messages-column'),
    prose: cs('.message.assistant .prose'), userBubble: cs('.user-bubble'),
    composer: cs('.composer-box'), toolbar: cs('.composer-toolbar'),
    proseText: (prose?.innerText || '').slice(0, 100),
  }
})
out.darkTyped = typed
console.log('DARK:', JSON.stringify(out.dark, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.concat(dcon.events).slice(0, 25), null, 1))
console.log('RESULT:', JSON.stringify(out, null, 1))
await browser.close()
