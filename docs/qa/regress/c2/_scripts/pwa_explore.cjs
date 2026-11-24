const { launch, PWA, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 1280, height: 860 }, deviceScaleFactor: 1 })
  const page = await ctx.newPage()
  const allLogs = []
  page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') allLogs.push(`console.${m.type()}: ${m.text()}`) })
  page.on('pageerror', (e) => allLogs.push(`pageerror: ${e.message}`))

  await page.goto(PWA + '/', { waitUntil: 'domcontentloaded' }).catch((e) => console.log('GOTO_ERR', e.message))
  await page.waitForTimeout(4000)
  console.log('URL0:', page.url())

  // login if needed
  const email = page.locator('input[type=email]')
  if (await email.count()) {
    await email.first().fill(USER)
    await page.locator('input[type=password]').first().fill(PASS)
    await page.locator('button[type=submit]').first().click().catch(() => {})
    await page.waitForTimeout(5000)
  }
  console.log('URL1:', page.url())
  const beforeTxt = (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim()
  console.log('BEFORE:', beforeTxt.slice(0, 200))
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-home.png` })

  // open avatar menu
  const cands = await page.evaluate(() => {
    const vis = (n) => { const r = n.getBoundingClientRect(); return r.width > 0 && r.height > 0 }
    return [...document.querySelectorAll('header button, .avatar, [class*=avatar], [aria-label*=账号], [aria-label*=用户], [class*=user-menu], [class*=account]')]
      .filter(vis)
      .map((e) => ({ tag: e.tagName.toLowerCase(), cls: (e.className || '').toString().slice(0, 60), aria: e.getAttribute('aria-label'), txt: (e.innerText || '').trim().slice(0, 20), box: (() => { const r = e.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] })() }))
  })
  console.log('AVATAR_CANDS:', JSON.stringify(cands, null, 2))

  const avatar = page.locator('header [class*=avatar], .avatar, [aria-label*=账号]').first()
  if (await avatar.count()) {
    await avatar.click()
    await page.waitForTimeout(1200)
  }
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-menu.png` })
  const menuTxt = (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim()
  console.log('MENU_TXT_TAIL:', menuTxt.slice(-400))
  const logoutCount = await page.getByText('退出登录').count()
  console.log('LOGOUT_COUNT:', logoutCount)
  console.log('LOGS:\n' + allLogs.join('\n'))
  await b.close()
})()
