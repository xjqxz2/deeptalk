import { launch, trackConsole, shot, sleep, ACCOUNT } from './lib.mjs'

const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')

await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2500)
console.log('after login url:', page.url())

const result = {}

// ---- 路径 A：侧栏账号卡菜单（BUG 修复目标路径）----
await page.locator('button.user-card').click()
await page.waitForSelector('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu', { timeout: 5000 })
const menuTextA = await page.locator('.ant-dropdown:not(.ant-dropdown-hidden)').innerText()
console.log('menu A:', JSON.stringify(menuTextA))
await shot(page, '_b2792968-menu')
await page.locator('.ant-dropdown:not(.ant-dropdown-hidden)').getByText('退出登录', { exact: true }).click()
await page.waitForTimeout(900)
const modalA = await page.locator('.ant-modal-confirm').count()
const modalTitleA = modalA ? (await page.locator('.ant-modal-confirm-title').innerText().catch(()=> '')) : ''
const modalBodyA = modalA ? (await page.locator('.ant-modal-confirm-content').innerText().catch(()=> '')) : ''
result.sidebarMenu = { modal: modalA, title: modalTitleA, body: modalBodyA, url: page.url() }
await shot(page, 'BUG-B2792968-确认弹窗')
console.log('A modal:', JSON.stringify(result.sidebarMenu))

// 点取消
if (modalA) {
  await page.locator('.ant-modal-confirm-btns button').nth(0).click()
  await page.waitForTimeout(900)
}
result.afterCancel = {
  url: page.url(),
  stillChat: await page.locator('.chat-shell, .chat-sidebar').count() > 0,
  userCardVisible: await page.locator('button.user-card').isVisible().catch(() => false),
  modalGone: (await page.locator('.ant-modal-confirm').count()) === 0,
}
await shot(page, 'BUG-B2792968-取消后仍在聊天页')
console.log('after cancel:', JSON.stringify(result.afterCancel))

// ---- 路径 B：顶栏头像菜单（同样标注 aria-label=账号菜单）----
await page.locator('button.avatar').click()
await page.waitForTimeout(700)
const menuTextB = await page.locator('.ant-dropdown:not(.ant-dropdown-hidden)').innerText().catch(()=> '')
console.log('menu B:', JSON.stringify(menuTextB))
result.topbarMenu = { text: menuTextB }
console.log('RESULT:', JSON.stringify(result, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!e.text.includes('/auth/me')), null, 1))
await browser.close()
