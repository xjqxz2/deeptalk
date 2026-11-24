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

// ---- 多行输入框变高 ----
const ta = page.locator('textarea.composer-input')
out.taHeight0 = (await ta.boundingBox())?.height
await ta.click()
await ta.fill('第一行\n第二行\n第三行\n第四行')
await page.waitForTimeout(700)
out.taHeightMulti = (await ta.boundingBox())?.height
out.taScrollHeight = await ta.evaluate(el => el.scrollHeight)
await shot(page, 'BUG-236528F2-多行输入框变高')
await ta.fill('')
await page.waitForTimeout(400)

// ---- 停止生成 → 已停止生成 ----
await ta.fill('详细解释一下 SSE 和 WebSocket 的区别，至少写 800 字')
await ta.press('Enter')
let started = false
for (let i = 0; i < 40; i++) {
  await page.waitForTimeout(500)
  if (await page.locator('.stop-btn').count()) { started = true; break }
}
out.stopBtnFound = started
if (started) {
  await page.waitForTimeout(2500)   // 让它先输出一点内容
  await page.locator('.stop-btn').click()
  await page.waitForTimeout(2500)
  out.stopStats = await page.locator('.message.assistant').last().locator('.stats').innerText().catch(()=> '')
  out.stoppedMsgClass = await page.locator('.message.assistant').last().getAttribute('class')
  await shot(page, 'BUG-236528F2-已停止生成')
}
console.log('STOP:', JSON.stringify({started, stats: out.stopStats, cls: out.stoppedMsgClass}))

// ---- 删除会话确认框（只在抽屉里打开确认框，然后取消）----
await page.locator('button.menu-btn').click()
await page.waitForTimeout(900)
await page.locator('.history-item .more').first().click()
await page.waitForTimeout(900)
out.sheet = await page.locator('.uni-actionsheet__cell').allInnerTexts().catch(()=>[])
console.log('SHEET:', JSON.stringify(out.sheet))
const del = page.locator('.uni-actionsheet__cell', { hasText: '删除' }).first()
if (await del.count()) {
  await del.click()
  await page.waitForTimeout(1200)
  out.modal = await page.evaluate(() => {
    const m = document.querySelector('.uni-modal, [class*="uni-modal"]')
    return m ? { cls: m.className, title: (m.querySelector('.uni-modal__hd, [class*="hd"]')||{}).innerText || '', body: (m.querySelector('.uni-modal__bd, [class*="bd"]')||{}).innerText || '', text: m.innerText.slice(0,200) } : null
  })
  await shot(page, 'BUG-236528F2-删除确认框')
  // 取消
  const cancel = page.locator('.uni-modal__btn, [class*="modal"] button').filter({ hasText: /取消|Cancel/ }).first()
  if (await cancel.count()) await cancel.click().catch(()=>{})
  await page.waitForTimeout(600)
}
console.log('DELETE MODAL:', JSON.stringify(out.modal))
console.log('TEXTAREA:', JSON.stringify({ h0: out.taHeight0, multi: out.taHeightMulti, scroll: out.taScrollHeight }))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,15), null, 1))
await browser.close()
