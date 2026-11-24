const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'drawer2')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(1200)

  const html = await page.evaluate(() => {
    const sf = document.querySelector('input.search-field')
    let n = sf
    for (let i = 0; i < 6 && n && n.parentElement; i++) n = n.parentElement
    return n ? n.outerHTML.slice(0, 4000) : 'not found'
  })
  console.log('DRAWER_HTML:\n' + html)
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
