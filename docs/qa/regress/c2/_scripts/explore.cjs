const { launch, wire, logs, H5 } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'explore')
  await page.goto(H5, { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(2500)
  console.log('URL:', page.url())
  const info = await page.evaluate(() => {
    const pick = (el) => ({
      tag: el.tagName.toLowerCase(),
      type: el.getAttribute('type'),
      ph: el.getAttribute('placeholder'),
      cls: el.className && el.className.toString().slice(0, 120),
      text: (el.innerText || '').trim().slice(0, 60),
    })
    return {
      inputs: [...document.querySelectorAll('input,textarea')].map(pick),
      buttons: [...document.querySelectorAll('button, uni-button, .uni-btn, [class*=btn]')].map(pick).slice(0, 40),
      bodyText: (document.body.innerText || '').slice(0, 1200),
      html: document.getElementById('app') ? document.getElementById('app').innerHTML.length : -1,
    }
  })
  console.log(JSON.stringify(info, null, 2))
  console.log('LOGS:', logs.join('\n'))
  await b.close()
})()
