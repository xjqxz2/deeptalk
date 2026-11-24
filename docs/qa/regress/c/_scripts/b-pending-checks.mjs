import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1600, height: 1000 }, acceptDownloads: true })
const page = await ctx.newPage()
const con = trackConsole(page, 'admin')
const out = {}
await page.goto('https://deeptalk.peon.top/admin/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1500)
await page.locator('input[placeholder="admin@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button[type="submit"]').first().click()
await page.waitForTimeout(4000)
out.afterLogin = page.url()

// ---- 概览：时间范围切换 ----
await page.goto('https://deeptalk.peon.top/admin/overview', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)
out.overview = await page.evaluate(() => ({
  radios: [...document.querySelectorAll('.ant-radio-button-wrapper')].map(e => e.innerText.trim()),
  seg: [...document.querySelectorAll('.ant-segmented-item')].map(e => e.innerText.trim()),
  sub: [...document.querySelectorAll('p')].map(e => e.innerText.trim()).filter(t => /最近/.test(t)),
}))
out.rangeSwitch = (await page.locator('.ant-radio-button-wrapper').count()) > 0
if (out.rangeSwitch) {
  await page.locator('.ant-radio-button-wrapper', { hasText: '7' }).first().click()
  await page.waitForTimeout(1800)
  out.overviewAfter7 = await page.evaluate(() => [...document.querySelectorAll('p')].map(e => e.innerText.trim()).filter(t => /最近/.test(t)))
}
await shot(page, 'pending-概览时间范围切换')

// ---- 用量统计：时间范围 / 导出 / 表头排序 ----
await page.goto('https://deeptalk.peon.top/admin/usage', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)
out.usage = await page.evaluate(() => ({
  buttons: [...document.querySelectorAll('button')].map(b => b.innerText.trim()).filter(Boolean),
  headers: [...document.querySelectorAll('.usage-table th')].map(e => e.innerText.trim()),
  sortable: document.querySelectorAll('.usage-table th.sortable, .usage-table th.sort-mark, .usage-table th span.sort-mark').length,
  rows: document.querySelectorAll('.usage-table tbody tr').length,
}))
out.exportBtn = await page.locator('button', { hasText: '导出' }).count()
await shot(page, 'pending-用量导出与表头排序')

// 导出点击 → 是否触发下载/提示
if (out.exportBtn) {
  const dl = page.waitForEvent('download', { timeout: 8000 }).catch(() => null)
  await page.locator('button', { hasText: '导出' }).first().click()
  const download = await dl
  out.download = download ? download.suggestedFilename() : null
  out.toastAfterExport = await page.evaluate(() => [...document.querySelectorAll('.ant-message-notice-content')].map(e => e.innerText.trim()))
  if (download) await download.saveAs('/home/xjqxz/Workplace/deeptalk/docs/qa/regress/c/_usage-export.csv').catch(()=>{})
  await page.waitForTimeout(800)
  await shot(page, 'pending-用量导出点击后')
}

// 表头排序：点击 tokens 列，比较第一行
const before = await page.evaluate(() => [...document.querySelectorAll('.usage-table tbody tr')].map(r => r.innerText.replace(/\s+/g,' ').slice(0,40)).slice(0,3))
const tokenTh = page.locator('.usage-table th', { hasText: 'tokens' }).first()
if (await tokenTh.count()) {
  await tokenTh.click(); await page.waitForTimeout(1000)
  const after1 = await page.evaluate(() => ({ rows: [...document.querySelectorAll('.usage-table tbody tr')].map(r => r.innerText.replace(/\s+/g,' ').slice(0,40)).slice(0,3), marks: [...document.querySelectorAll('.usage-table .sort-mark')].map(e=>e.textContent.trim()) }))
  await tokenTh.click(); await page.waitForTimeout(1000)
  const after2 = await page.evaluate(() => ({ rows: [...document.querySelectorAll('.usage-table tbody tr')].map(r => r.innerText.replace(/\s+/g,' ').slice(0,40)).slice(0,3), marks: [...document.querySelectorAll('.usage-table .sort-mark')].map(e=>e.textContent.trim()) }))
  out.sort = { before, after1, after2 }
  await shot(page, 'pending-用量表头排序点击后')
}
console.log('RESULT:', JSON.stringify(out, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e => !/auth\/me/.test(e.text)).slice(0, 20), null, 1))
await browser.close()
