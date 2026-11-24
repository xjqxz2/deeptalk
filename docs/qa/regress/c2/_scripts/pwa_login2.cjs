const { launch, PWA, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 1280, height: 900 } })
  const page = await ctx.newPage()
  const allLogs = []
  page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') allLogs.push(`console.${m.type()}: ${m.text()}`) })
  page.on('pageerror', (e) => allLogs.push(`pageerror: ${e.message}`))
  page.on('response', async (r) => {
    if (/\/api\/(auth|login|session|me)/.test(r.url())) allLogs.push(`HTTP ${r.status()} ${r.request().method()} ${r.url()}`)
  })

  await page.goto(PWA + '/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(2500)

  // ---- PWA forgot-password (supplementary to uni item 4)
  await page.locator('button.text-link', { hasText: '忘记密码' }).first().click().catch((e) => console.log('FP_CLICK_ERR', e.message))
  await page.waitForTimeout(1500)
  const fpState = await page.evaluate(() => {
    const vis = (n) => { const r = n.getBoundingClientRect(); const cs = getComputedStyle(n); return r.width > 0 && r.height > 0 && cs.display !== 'none' && cs.visibility !== 'hidden' }
    return {
      modals: [...document.querySelectorAll('.ant-modal, .ant-modal-wrap, .ant-message, .ant-notification, [role=dialog], .ant-popover')].filter(vis).map((n) => ({ cls: (n.className || '').toString().slice(0, 70), txt: (n.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 200) })),
      bodyTail: (document.body.innerText || '').replace(/\s+/g, ' ').trim().slice(-260),
    }
  })
  console.log('PWA_FORGOT:', JSON.stringify(fpState, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-forgot.png` })
  // dismiss any modal
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(600)

  // ---- login
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.waitForTimeout(300)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(6000)
  console.log('URL_AFTER_LOGIN:', page.url())
  console.log('BODY_HEAD:', (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim().slice(0, 200))
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-after-login.png` })

  if (/\/login/.test(page.url())) {
    console.log('LOGIN_FAILED_LOGS:\n' + allLogs.join('\n'))
    await b.close()
    return
  }

  // ---- avatar menu
  const avatarInfo = await page.evaluate(() => {
    const vis = (n) => { const r = n.getBoundingClientRect(); return r.width > 0 && r.height > 0 }
    return [...document.querySelectorAll('header button, header [class*=avatar], header img, [class*=avatar], [aria-label*=账号], [aria-label*=用户]')].filter(vis)
      .map((e) => ({ tag: e.tagName.toLowerCase(), cls: (e.className || '').toString().slice(0, 60), aria: e.getAttribute('aria-label'), txt: (e.innerText || '').trim().slice(0, 20), box: (() => { const r = e.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] })() }))
  })
  console.log('AVATARS:', JSON.stringify(avatarInfo, null, 2))
  const target = avatarInfo.find((a) => /avatar|账号|用户/.test((a.cls || '') + (a.aria || '')))
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-dashboard.png` })
  console.log('PICKED:', JSON.stringify(target))
  await b.close()
})()
