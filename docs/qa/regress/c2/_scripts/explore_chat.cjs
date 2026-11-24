const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

async function login(page) {
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  return page.url()
}

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'chat-explore')
  console.log('AFTER_LOGIN_URL:', await login(page))
  console.log('BODY:', JSON.stringify((await page.evaluate(() => document.body.innerText)).slice(0, 800)))

  const dom = await page.evaluate(() => {
    const pick = (el) => {
      const r = el.getBoundingClientRect()
      return {
        tag: el.tagName.toLowerCase(),
        cls: (el.className || '').toString().slice(0, 90),
        ph: el.getAttribute('placeholder'),
        aria: el.getAttribute('aria-label'),
        txt: (el.innerText || '').trim().slice(0, 40),
        box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      }
    }
    return {
      textareas: [...document.querySelectorAll('textarea')].map(pick),
      inputs: [...document.querySelectorAll('input')].map(pick),
      buttons: [...document.querySelectorAll('button')].map(pick),
      clickables: [...document.querySelectorAll('[role=button],[class*=btn],[class*=icon],[class*=avatar],[class*=menu],[class*=drawer],[class*=header]')].map(pick).slice(0, 40),
    }
  })
  console.log('DOM:', JSON.stringify(dom, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/_chat-home.png`, fullPage: true })
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
