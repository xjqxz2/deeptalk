const { launch, PWA, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 1280, height: 900 } })
  const page = await ctx.newPage()
  const allLogs = []
  page.on('console', (m) => { if (m.type() === 'error') allLogs.push(`console.error: ${m.text()}`) })
  page.on('pageerror', (e) => allLogs.push(`pageerror: ${e.message}`))
  await page.goto(PWA + '/login', { waitUntil: 'domcontentloaded' }).catch(() => {})
  await page.waitForTimeout(3500)
  const dom = await page.evaluate(() => {
    const pick = (el) => {
      const r = el.getBoundingClientRect()
      return { tag: el.tagName.toLowerCase(), type: el.getAttribute('type'), cls: (el.className || '').toString().slice(0, 70), ph: el.getAttribute('placeholder'), txt: (el.innerText || '').trim().slice(0, 24), box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)] }
    }
    return { inputs: [...document.querySelectorAll('input')].map(pick), buttons: [...document.querySelectorAll('button')].map(pick) }
  })
  console.log('LOGIN_DOM:', JSON.stringify(dom, null, 2))
  console.log('LOGS:', allLogs.join('\n'))
  await b.close()
})()
