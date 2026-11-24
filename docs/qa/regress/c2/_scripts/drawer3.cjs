const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'drawer3')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(1200)

  const info = await page.evaluate(() => {
    const sf = document.querySelector('input.search-field')
    const chain = []
    let n = sf
    while (n && chain.length < 10) {
      chain.push({ tag: n.tagName.toLowerCase(), cls: (n.className || '').toString(), kids: n.children.length })
      n = n.parentElement
    }
    // find the nearest ancestor with more than 3 children (list container candidate)
    const cand = [...document.querySelectorAll('[class*=drawer],[class*=side],[class*=session],[class*=list],[class*=conv]')]
      .map((e) => ({ cls: (e.className || '').toString().slice(0, 70), kids: e.children.length, txt: (e.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 120) }))
      .slice(0, 20)
    return { chain, cand }
  })
  console.log(JSON.stringify(info, null, 2))
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
