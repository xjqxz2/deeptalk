import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const UNI = 'http://127.0.0.1:5331/'
const browser = await launch()
const results = {}

// --- 正常（浅色）登录页：忘记密码点击 ---
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'light' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
await page.goto(UNI, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2500)
await shot(page, '_uni-login-light')
const beforeHTML = await page.evaluate(() => document.body.innerHTML.length)
await page.locator('.field-link, span.field-link').first().click({ force: true }).catch(e => console.log('click err', e.message.slice(0,80)))
await page.waitForTimeout(1500)
const after = await page.evaluate(() => ({
  htmlLen: document.body.innerHTML.length,
  uniModal: [...document.querySelectorAll('.uni-modal, .uni-modal-box, [class*="modal"]')].map(e => e.innerText.trim().slice(0,120)),
  toast: [...document.querySelectorAll('.uni-toast, [class*="toast"]')].map(e => e.innerText.trim().slice(0,80)),
  bodyText: document.body.innerText.slice(0, 300),
}))
results.forgot = { beforeHTML, after, changed: after.htmlLen !== beforeHTML }
console.log('FORGOT:', JSON.stringify(results.forgot, null, 1))
await shot(page, '_uni-forgot-click')

// --- 登录 ---
await page.locator('input[type="email"]').fill(ACCOUNT.email)
await page.locator('input[type="password"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForTimeout(5000)
results.afterLogin = { url: page.url(), text: (await page.evaluate(() => document.body.innerText)).slice(0, 300) }
console.log('AFTER LOGIN:', JSON.stringify(results.afterLogin))
await shot(page, '_uni-chat-light')
await ctx.close()

// --- 深色：登录页 ---
const dctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'dark' })
const dpage = await dctx.newPage()
await dpage.emulateMedia({ colorScheme: 'dark' })
await dpage.goto(UNI, { waitUntil: 'domcontentloaded' })
await dpage.waitForTimeout(2500)
await shot(dpage, 'BUG-9698E1ED-深色登录页')
results.darkLogin = await dpage.evaluate(() => {
  const cs = (sel) => { const e = document.querySelector(sel); if (!e) return null; const s = getComputedStyle(e); return { color: s.color, bg: s.backgroundColor, bgImage: s.backgroundImage.slice(0,80) } }
  return {
    body: cs('body'), shell: cs('.auth-shell'), panel: cs('.auth-panel'),
    title: cs('h1'), subtitle: cs('.subtitle'), label: cs('.field-label'),
    input: cs('input[type="email"]'), submit: cs('.submit-btn'),
    text: document.body.innerText.slice(0, 200),
  }
})
console.log('DARK LOGIN:', JSON.stringify(results.darkLogin, null, 1))
await dctx.close()
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,20), null, 1))
await browser.close()
