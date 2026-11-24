const { launch, H5, SHOT_DIR } = require('./lib.cjs')
;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2, colorScheme: 'light' })
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push('pageerror: ' + e.message))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('console.error: ' + m.text()) })
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1800)
  const before = (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim()
  await page.getByText('忘记密码？').first().click()
  await page.waitForTimeout(1500)
  const st = await page.evaluate(() => {
    const vis = (n) => { const r = n.getBoundingClientRect(); const cs = getComputedStyle(n); return r.width > 0 && r.height > 0 && cs.display !== 'none' && Number(cs.opacity) > 0.01 }
    return {
      overlays: [...document.querySelectorAll('uni-modal,.uni-modal,.uni-mask,uni-toast,.uni-toast,[class*=modal],[class*=toast],[class*=dialog]')].filter(vis).map((n) => ({ cls: (n.className||'').toString().slice(0,60), txt: (n.innerText||'').trim().slice(0,150) })),
      changed: null,
    }
  })
  const after = (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim()
  console.log('LIGHT_MODE_overlays:', JSON.stringify(st.overlays))
  console.log('BODY_UNCHANGED:', before === after)
  console.log('ERRS:', errs.join(' | ') || 'none')
  await page.screenshot({ path: `${SHOT_DIR}/4-失败-浅色同样无反应.png` })
  await b.close()
})()
