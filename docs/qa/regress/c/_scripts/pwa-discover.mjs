import { launch, trackConsole, shot, sleep, ACCOUNT } from './lib.mjs'

const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')

async function login(page) {
  await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const email = page.locator('input[placeholder="you@company.com"]')
  await email.fill(ACCOUNT.email)
  await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
  await page.locator('button.submit-btn').click()
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
  await page.waitForTimeout(3000)
}
await login(page)
console.log('URL after login:', page.url())
await shot(page, '_discover-chat')
const info = await page.evaluate(() => {
  const btns = [...document.querySelectorAll('button')].map(b => ({
    cls: b.className, aria: b.getAttribute('aria-label'), text: (b.textContent||'').trim().slice(0,40)
  }))
  return {
    title: document.title,
    buttons: btns,
    textareas: [...document.querySelectorAll('textarea')].map(t => ({aria: t.getAttribute('aria-label'), ph: t.placeholder})),
    sidebarConvCount: document.querySelectorAll('.history-item').length,
    bodyText: document.body.innerText.slice(0, 1200),
  }
})
console.log(JSON.stringify(info, null, 2))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,30), null, 1))
await browser.close()
