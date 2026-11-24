const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'reload')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)

  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(1000)
  const first = page.locator('.history-group [class*=item], .history-group li, .history-group > div > *').first()
  const items = await page.evaluate(() =>
    [...document.querySelectorAll('.history-group > *')].slice(0, 6).map((e) => ({ cls: (e.className || '').toString().slice(0, 40), txt: (e.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 40) }))
  )
  console.log('HISTORY_HEAD:', JSON.stringify(items, null, 2))
  const target = page.locator('aside.chat-sidebar').getByText('SSE 服务器推送技术简介', { exact: true }).first()
  console.log('TARGET_COUNT:', await page.locator('aside.chat-sidebar').getByText('SSE 服务器推送技术简介', { exact: true }).count())
  await target.click()
  await page.waitForTimeout(3000)

  const rows = await page.evaluate(() => {
    const col = document.querySelector('.messages-column')
    return [...(col ? col.children : [])].filter((e) => /message/.test((e.className || '').toString())).map((e) => {
      const cls = (e.className || '').toString()
      const body = e.querySelector('.dt-prose, .user-bubble')
      const t = (body?.innerText || '').replace(/\s+/g, ' ').trim()
      return { role: /assistant/.test(cls) ? 'A' : 'U', status: /streaming/.test(cls) ? 'streaming' : 'complete', len: t.length, head: t.slice(0, 50) }
    })
  })
  console.log('RELOADED_ROWS:', JSON.stringify(rows, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/1-补充-刷新后第一轮内容仍在.png` })
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
