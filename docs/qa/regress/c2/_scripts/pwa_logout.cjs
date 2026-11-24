const { launch, PWA, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 1280, height: 900 } })
  const page = await ctx.newPage()
  const allLogs = []
  page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') allLogs.push(`console.${m.type()}: ${m.text()}`) })
  page.on('pageerror', (e) => allLogs.push(`pageerror: ${e.message}`))
  page.on('response', (r) => { if (/\/api\//.test(r.url())) allLogs.push(`HTTP ${r.status()} ${r.request().method()} ${r.url().slice(0, 110)}`) })

  await page.goto(PWA + '/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(2500)
  await page.locator('input[placeholder="you@company.com"]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.waitForTimeout(400)
  await Promise.all([
    page.locator('button.submit-btn').click(),
    page.waitForTimeout(8000),
  ])
  const afterLogin = page.url()
  console.log('URL_AFTER_LOGIN:', afterLogin)
  console.log('BODY_HEAD:', (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim().slice(0, 160))
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-after-login.png` })
  if (/\/login/.test(afterLogin)) {
    console.log('LOGIN_FAILED. LOGS:\n' + allLogs.join('\n'))
    await b.close()
    return
  }

  // avatar menu
  const avatars = await page.evaluate(() => {
    const vis = (n) => { const r = n.getBoundingClientRect(); return r.width > 0 && r.height > 0 }
    return [...document.querySelectorAll('header button, header [class*=avatar], header [class*=user], [class*=avatar], [aria-label*=账号], [aria-label*=用户], [class*=account]')].filter(vis)
      .map((e) => ({ tag: e.tagName.toLowerCase(), cls: (e.className || '').toString().slice(0, 60), aria: e.getAttribute('aria-label'), txt: (e.innerText || '').trim().slice(0, 16), box: (() => { const r = e.getBoundingClientRect(); return [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] })() }))
  })
  console.log('AVATARS:', JSON.stringify(avatars, null, 2))

  let clicked = false
  for (const a of avatars) {
    if (/avatar|账号|用户|account|user/i.test((a.cls || '') + ' ' + (a.aria || ''))) {
      const [x, y, w, h] = a.box
      await page.mouse.click(x + w / 2, y + h / 2)
      clicked = true
      console.log('CLICKED_AVATAR', JSON.stringify(a))
      break
    }
  }
  if (!clicked) {
    // fallback: last header button
    const btns = page.locator('header button')
    const n = await btns.count()
    console.log('FALLBACK header buttons', n)
    if (n) await btns.nth(n - 1).click()
  }
  await page.waitForTimeout(1500)
  await page.screenshot({ path: `${SHOT_DIR}/_pwa-menu.png` })
  const menuTxt = (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim()
  console.log('MENU_HAS_LOGOUT:', /退出登录/.test(menuTxt), '| tail:', menuTxt.slice(-240))
  const lo = page.getByText('退出登录').first()
  console.log('LOGOUT_COUNT:', await page.getByText('退出登录').count())

  if (await page.getByText('退出登录').count()) {
    const urlBefore = page.url()
    await lo.click()
    await page.waitForTimeout(1800)
    const st = await page.evaluate(() => {
      const vis = (n) => { const r = n.getBoundingClientRect(); const cs = getComputedStyle(n); return r.width > 0 && r.height > 0 && cs.display !== 'none' && Number(cs.opacity) > 0.01 }
      return {
        dialogs: [...document.querySelectorAll('.ant-modal, .ant-modal-wrap, .ant-modal-confirm, [role=dialog], .ant-popconfirm, .ant-popover, .uni-modal, [class*=modal]')].filter(vis)
          .map((n) => ({ cls: (n.className || '').toString().slice(0, 60), txt: (n.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 200) })),
        url: location.href,
      }
    })
    console.log('AFTER_LOGOUT_CLICK:', JSON.stringify(st, null, 2))
    await page.screenshot({ path: `${SHOT_DIR}/6-退出登录确认框.png` })

    // click cancel
    const btns = await page.evaluate(() => [...document.querySelectorAll('.ant-modal-confirm-btns button')].map((b) => (b.innerText || '').replace(/\s+/g, '')))
    console.log('CONFIRM_BTNS:', JSON.stringify(btns))
    const cancel = page.locator('.ant-modal-confirm-btns button').first()
    const cancelCount = await cancel.count()
    console.log('CANCEL_COUNT:', cancelCount)
    if (cancelCount) {
      await cancel.click()
      await page.waitForTimeout(2000)
      const st2 = await page.evaluate(() => ({ url: location.href, bodyHead: (document.body.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 120) }))
      console.log('AFTER_CANCEL:', JSON.stringify(st2, null, 2))
      console.log('STILL_ON_SAME_PAGE:', st2.url === st.url)
      await page.screenshot({ path: `${SHOT_DIR}/6-取消后仍在原页.png` })
    }
  }
  console.log('LOGS:\n' + allLogs.join('\n'))
  await b.close()
})()
