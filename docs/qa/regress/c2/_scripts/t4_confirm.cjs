const { launch, H5, SHOT_DIR } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2, colorScheme: 'dark' })
  const page = await ctx.newPage()
  const allLogs = []
  page.on('console', (m) => allLogs.push(`console.${m.type()}: ${m.text()}`))
  page.on('pageerror', (e) => allLogs.push(`pageerror: ${e.message}`))
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1800)

  // control: does a click work at all here?
  await page.getByText('立即注册').first().click()
  await page.waitForTimeout(500)
  const afterSwitch = (await page.evaluate(() => document.body.innerText)).slice(0, 200)
  console.log('AFTER_切换注册:', JSON.stringify(afterSwitch))
  await page.getByText('去登录').first().click()
  await page.waitForTimeout(500)
  console.log('AFTER_回到登录:', JSON.stringify((await page.evaluate(() => document.body.innerText)).slice(0, 120)))

  // instrument: count mutations + click events reaching document
  await page.evaluate(() => {
    window.__mut = 0
    window.__clicks = []
    new MutationObserver((rs) => { window.__mut += rs.length }).observe(document.body, { childList: true, subtree: true, attributes: true, characterData: true })
    document.addEventListener('click', (e) => {
      const t = e.target
      window.__clicks.push((t.className && t.className.toString().slice(0, 60)) + '|' + (t.innerText || '').trim().slice(0, 20))
    }, true)
  })
  const fp = page.getByText('忘记密码？').first()
  const box = await fp.boundingBox()
  console.log('FORGOT_BOX:', JSON.stringify(box))
  await fp.click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${SHOT_DIR}/4-忘记密码-点击后300ms.png` })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: `${SHOT_DIR}/4-忘记密码弹窗.png` })
  const probe = await page.evaluate(() => ({
    mut: window.__mut,
    clicks: window.__clicks,
    overlays: [...document.querySelectorAll('uni-modal,.uni-modal,.uni-mask,uni-toast,.uni-toast,uni-actionsheet,uni-loading,uni-picker,[class*=modal],[class*=toast],[class*=dialog],[class*=popup],[class*=overlay]')]
      .map((n) => ({ cls: n.className.toString().slice(0, 80), disp: getComputedStyle(n).display, vis: getComputedStyle(n).visibility, op: getComputedStyle(n).opacity, txt: (n.innerText || '').trim().slice(0, 200) })),
    bodyLen: document.body.innerText.length,
    hasCancel: /取消/.test(document.body.innerText),
    hasOk: /确定/.test(document.body.innerText),
  }))
  console.log('PROBE:', JSON.stringify(probe, null, 2))
  console.log('ALL_LOGS:\n' + allLogs.join('\n'))
  await b.close()
})()
