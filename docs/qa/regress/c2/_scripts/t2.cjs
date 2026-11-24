const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'item2')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(1000)

  const skeleton = await page.evaluate(() => {
    const body = document.querySelector('.sidebar-body')
    return [...body.children].map((c) => ({
      cls: (c.className || '').toString(),
      kids: [...c.children].map((k) => ({ cls: (k.className || '').toString().slice(0, 60), txt: (k.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 50) })).slice(0, 6),
    }))
  })
  console.log('SKELETON:', JSON.stringify(skeleton, null, 2))

  const sf = page.locator('input.search-field')
  async function snap(label) {
    const s = await page.evaluate(() => {
      const body = document.querySelector('.sidebar-body')
      const items = [...body.querySelectorAll('[class*=session-item],[class*=conv-item],[class*=chat-item],[class*=history-item],[class*=item]')]
        .filter((e) => (e.innerText || '').trim().length > 0)
      return {
        items: items.length,
        titles: items.map((e) => (e.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 30)).slice(0, 30),
        bodyText: (body.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 400),
        emptyEls: [...document.querySelectorAll('[class*=empty]')].map((e) => ({ cls: (e.className || '').toString().slice(0, 50), txt: (e.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 80) })),
      }
    })
    console.log(`[${label}]`, JSON.stringify(s, null, 2))
    return s
  }

  await snap('INITIAL')

  await sf.click()
  await sf.fill('zzzz')
  await page.waitForTimeout(1000)
  const z = await snap('SEARCH_zzzz')
  await page.screenshot({ path: `${SHOT_DIR}/2-抽屉搜索-zzzz空态.png` })

  await sf.fill('')
  await page.waitForTimeout(600)
  await sf.fill('SSE')
  await page.waitForTimeout(1000)
  const s = await snap('SEARCH_SSE')
  await page.screenshot({ path: `${SHOT_DIR}/2-抽屉搜索-SSE过滤.png` })

  console.log('SUMMARY:', JSON.stringify({ zzzz_items: z.items, zzzz_empty: z.bodyText, sse_items: s.items, sse_titles: s.titles }, null, 2))
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
