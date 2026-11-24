const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'drawer')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  console.log('URL:', page.url())

  // ---- item 5 first: multiline composer
  const ta = page.locator('textarea.composer-input')
  const before = await ta.evaluate((el) => ({ ch: el.clientHeight, sh: el.scrollHeight, oh: el.offsetHeight, styleH: el.style.height, maxH: getComputedStyle(el).maxHeight }))
  console.log('TA_BEFORE:', JSON.stringify(before))
  await ta.click()
  await page.keyboard.type('第一行：验证多行输入框自动增高')
  await page.keyboard.press('Shift+Enter')
  await page.keyboard.type('第二行：内容不应被裁切')
  await page.keyboard.press('Shift+Enter')
  await page.keyboard.type('第三行：检查 clientHeight 与 scrollHeight')
  await page.keyboard.press('Shift+Enter')
  await page.keyboard.type('第四行：结束')
  await page.waitForTimeout(800)
  const after = await ta.evaluate((el) => ({ ch: el.clientHeight, sh: el.scrollHeight, oh: el.offsetHeight, styleH: el.style.height, maxH: getComputedStyle(el).maxHeight, value: el.value }))
  console.log('TA_AFTER:', JSON.stringify(after, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/5-多行输入框.png` })
  const taShot = `${SHOT_DIR}/5-多行输入框.png`
  console.log('SHOT5:', taShot)

  // clear composer (select all + delete) without sending
  await ta.click()
  await page.keyboard.press('Control+a')
  await page.keyboard.press('Backspace')
  await page.waitForTimeout(400)
  console.log('TA_CLEARED:', JSON.stringify(await ta.evaluate((el) => ({ ch: el.clientHeight, len: el.value.length }))))

  // ---- item 2: drawer
  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(1200)
  const drawerDom = await page.evaluate(() => {
    const pick = (el) => {
      const r = el.getBoundingClientRect()
      return {
        tag: el.tagName.toLowerCase(), cls: (el.className || '').toString().slice(0, 80),
        ph: el.getAttribute('placeholder'), aria: el.getAttribute('aria-label'),
        txt: (el.innerText || '').trim().slice(0, 40),
        box: [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)],
      }
    }
    const drawerSel = '[class*=drawer],[class*=sidebar],[class*=sider],[class*=panel],[class*=sheet]'
    return {
      inputs: [...document.querySelectorAll('input')].map(pick),
      buttons: [...document.querySelectorAll('button')].map(pick).slice(0, 25),
      drawerCandidates: [...document.querySelectorAll(drawerSel)].map(pick).slice(0, 12),
      bodyText: document.body.innerText.slice(0, 600),
    }
  })
  console.log('DRAWER:', JSON.stringify(drawerDom, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/_drawer-open.png`, fullPage: false })
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
