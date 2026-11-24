import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
await page.goto('http://127.0.0.1:5331/', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3000)
console.log('URL:', page.url())
await shot(page, '_uni-login')
const info = await page.evaluate(() => ({
  bodyText: document.body.innerText.slice(0, 800),
  inputs: [...document.querySelectorAll('input')].map(i => ({ type: i.type, ph: i.placeholder, cls: i.className })),
  clicks: [...document.querySelectorAll('[class*="submit"], [class*="field-link"], button')].map(e => ({ tag: e.tagName, cls: e.className, t: (e.textContent||'').trim().slice(0,30) })).slice(0, 20),
}))
console.log(JSON.stringify(info, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,20), null, 1))
await browser.close()
