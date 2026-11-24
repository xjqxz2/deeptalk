const { launch, wire, logs, H5, SHOT_DIR } = require('./lib.cjs')

const pick2 = (r) => `rgb(${r[0]},${r[1]},${r[2]})`
function lum(r) {
  const f = (c) => {
    c = c / 255
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * f(r[0]) + 0.7152 * f(r[1]) + 0.0722 * f(r[2])
}
function contrast(a, b) {
  const l1 = lum(a), l2 = lum(b)
  const [hi, lo] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (hi + 0.05) / (lo + 0.05)
}

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2, colorScheme: 'dark' })
  const page = await ctx.newPage()
  wire(page, 'dark-login')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(2000)

  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 15000 })
  const st = await email.evaluate((el) => {
    const cs = getComputedStyle(el)
    const parse = (v) => {
      const m = v.match(/rgba?\(([^)]+)\)/)
      if (!m) return null
      const p = m[1].split(/[,\s/]+/).filter(Boolean).map(Number)
      return p.slice(0, 3)
    }
    const probe = (sel) => {
      const n = document.querySelector(sel)
      if (!n) return null
      return parse(getComputedStyle(n).backgroundColor)
    }
    return {
      bg: cs.backgroundColor,
      color: cs.color,
      caret: cs.caretColor,
      border: cs.borderColor,
      bgRgb: parse(cs.backgroundColor),
      colorRgb: parse(cs.color),
      pageBg: getComputedStyle(document.body).backgroundColor,
      appBg: probe('#app'),
      wrapBg: probe('.native-auth-input') ,
      bodyDataTheme: document.body.getAttribute('data-theme') || document.documentElement.getAttribute('data-theme') || document.documentElement.className,
      rect: el.getBoundingClientRect().toJSON(),
    }
  })
  console.log('EMAIL_STYLE:', JSON.stringify(st, null, 2))
  if (st.bgRgb && st.colorRgb) {
    console.log('CONTRAST:', contrast(st.bgRgb, st.colorRgb).toFixed(2))
  }
  const inputFile = `${SHOT_DIR}/3-深色登录输入.png`
  await page.screenshot({ path: inputFile })
  const full = `${SHOT_DIR}/3-深色登录整页.png`
  await page.screenshot({ path: full, fullPage: true })
  console.log('SHOTS:', inputFile, full)

  // Item 4: forgot password
  const fp = page.getByText('忘记密码？', { exact: false }).first()
  const fpCount = await page.getByText('忘记密码？').count()
  console.log('FORGOT_EL_COUNT:', fpCount)
  await fp.click({ timeout: 10000 }).catch((e) => console.log('CLICK_ERR:', e.message))
  await page.waitForTimeout(1200)

  const after = await page.evaluate(() => {
    const vis = (n) => {
      const r = n.getBoundingClientRect()
      const cs = getComputedStyle(n)
      return r.width > 0 && r.height > 0 && cs.visibility !== 'hidden' && cs.display !== 'none' && Number(cs.opacity) > 0.01
    }
    const overlays = [...document.querySelectorAll('uni-modal, .uni-modal, .uni-mask, [class*=modal], [class*=toast], [class*=dialog], [class*=popup]')]
      .filter(vis)
      .map((n) => ({ cls: n.className.toString().slice(0, 100), text: (n.innerText || '').trim().slice(0, 300) }))
    return { overlays, bodyText: (document.body.innerText || '').slice(0, 1500) }
  })
  console.log('AFTER_CLICK:', JSON.stringify(after, null, 2))
  const fpFile = `${SHOT_DIR}/4-忘记密码弹窗.png`
  await page.screenshot({ path: fpFile })
  console.log('SHOT:', fpFile)
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
