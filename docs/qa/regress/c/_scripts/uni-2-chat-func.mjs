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
out.url = page.url()

// ---- 抽屉搜索 ----
await page.locator('button.menu-btn').click()
await page.waitForTimeout(900)
out.itemsBefore = await page.locator('.history-item').count()
await shot(page, '_uni-drawer-open')
const search = page.locator('input[aria-label="搜索对话"]')
await search.fill('SSE')
await page.waitForTimeout(1500)
out.itemsAfterKeyword = await page.locator('.history-item').count()
out.titlesAfterKeyword = await page.locator('.history-item .item-title').allInnerTexts()
await shot(page, 'BUG-F305D150-搜索过滤')
await search.fill('zzzz')
await page.waitForTimeout(1500)
out.itemsAfterZzzz = await page.locator('.history-item').count()
out.emptyText = await page.locator('.history-empty').innerText().catch(() => '')
await shot(page, 'BUG-F305D150-空态')
console.log('SEARCH:', JSON.stringify(out, null, 1))
await search.fill('')
await page.waitForTimeout(800)

// ---- 模型选择 ----
await page.locator('button.collapse-btn, .sidebar-mask').first().click()
await page.waitForTimeout(800)
out.modelBefore = (await page.locator('.model-trigger').innerText()).trim()
await page.locator('.model-trigger').click()
await page.waitForTimeout(1200)
out.popupVisible = await page.locator('.van-popup').isVisible().catch(() => false)
out.pickerTitle = await page.locator('.van-picker__title').innerText().catch(() => '')
out.options = await page.locator('.van-picker-column__item').allInnerTexts().catch(() => [])
await shot(page, 'BUG-96060A5E-模型选择器')
console.log('MODEL PICKER:', JSON.stringify({ popupVisible: out.popupVisible, title: out.pickerTitle, options: out.options.slice(0,10) }, null, 1))
// 选第二个模型并确认
if (out.options.length > 1) {
  await page.locator('.van-picker-column__item').nth(1).click()
  await page.waitForTimeout(600)
  await page.locator('.van-picker__confirm').click()
  await page.waitForTimeout(1500)
}
out.modelAfter = (await page.locator('.model-trigger').innerText().catch(() => '')).trim()
await shot(page, 'BUG-96060A5E-切换后模型')
console.log('MODEL AFTER:', JSON.stringify(out.modelBefore), '->', JSON.stringify(out.modelAfter))

// ---- 赞/踩 ----
const likeBtn = page.locator('.message.assistant .message-actions .action.icon').first()
out.likeCount = await page.locator('.message.assistant .message-actions .action.icon').count()
if (out.likeCount > 0) {
  await likeBtn.scrollIntoViewIfNeeded().catch(()=>{})
  await likeBtn.click({ force: true })
  await page.waitForTimeout(1500)
  out.likeClass = await likeBtn.getAttribute('class')
  out.activeCount = await page.locator('.message-actions .action.icon.active').count()
  await shot(page, 'BUG-96060A5E-点赞高亮')
}
out.errors = con.events.filter(e => e.kind === 'pageerror' || /onRate/.test(e.text))
console.log('RATE:', JSON.stringify({ likeCount: out.likeCount, likeClass: out.likeClass, active: out.activeCount, pageErrors: out.errors }, null, 1))
await ctx.close()

// ---- 深色聊天页 ----
const dctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'dark' })
const dpage = await dctx.newPage()
await dpage.emulateMedia({ colorScheme: 'dark' })
await dpage.goto(UNI, { waitUntil: 'domcontentloaded' })
await dpage.waitForTimeout(2500)
await dpage.locator('input[type="email"]').fill(ACCOUNT.email)
await dpage.locator('input[type="password"]').fill(ACCOUNT.password)
await dpage.locator('input[type="password"]').click()
await dpage.waitForTimeout(300)
await shot(dpage, 'BUG-9698E1ED-深色输入框文字')
const typedColor = await dpage.evaluate(() => { const i = document.querySelector('input[type="email"]'); const s = getComputedStyle(i); return { value: i.value, color: s.color, bg: s.backgroundColor, caret: s.caretColor } })
await dpage.locator('button.submit-btn').click()
await dpage.waitForTimeout(6000)
await shot(dpage, 'BUG-9698E1ED-深色聊天页')
out.darkChat = await dpage.evaluate(() => {
  const cs = (sel) => { const e = document.querySelector(sel); if (!e) return null; const s = getComputedStyle(e); return { color: s.color, bg: s.backgroundColor } }
  return {
    body: cs('body'), scroll: cs('.messages-scroll'), column: cs('.messages-column'),
    assistantProse: cs('.message.assistant .prose'), userBubble: cs('.user-bubble'),
    composer: cs('.composer-box'), input: cs('input, textarea'),
    text: (document.querySelector('.message.assistant .prose') || {}).innerText?.slice(0, 120) || '',
  }
})
out.typedColor = typedColor
console.log('DARK CHAT:', JSON.stringify(out.darkChat, null, 1))
await shot(dpage, '_uni-chat-dark')
console.log('CONSOLE:', JSON.stringify(con.events.slice(0, 25), null, 1))
console.log('FULL RESULT:', JSON.stringify(out, null, 1))
await browser.close()
