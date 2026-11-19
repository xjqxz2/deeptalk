/**
 * DeepTalk 冒烟测试（Playwright）。
 * 前置：后端已启动（默认 http://127.0.0.1:9821），演示数据已 seed。
 * 运行：node tests/smoke.mjs
 *
 * 说明：脚本会创建并删除一段测试对话，不会修改种子账号的额度与密码。
 */
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright-core'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '../..')
const base = process.env.BASE_URL || 'http://127.0.0.1:9821'
const executablePath =
  process.env.PW_CHROMIUM ||
  path.join(process.env.HOME || '', '.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell')

// 本地 QA 环境缺少系统库时，使用仓库内解压的依赖与字体配置
const deps = path.join(root, '.browser-deps')
process.env.LD_LIBRARY_PATH = [
  path.join(deps, 'root/usr/lib/x86_64-linux-gnu'),
  path.join(deps, 'root/usr/lib'),
  path.join(deps, 'root/lib/x86_64-linux-gnu'),
  process.env.LD_LIBRARY_PATH || '',
].join(':')
process.env.FONTCONFIG_FILE = path.join(deps, 'fonts.conf')
process.env.FONTCONFIG_PATH = deps

const results = []
const errors = []
const check = (name, condition) => {
  results.push({ name, ok: Boolean(condition) })
  if (!condition) throw new Error(`断言失败：${name}`)
}

const browser = await chromium.launch({ executablePath, args: ['--no-sandbox'] })
try {
  const cctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' })
  const page = await cctx.newPage()
  page.on('pageerror', (e) => errors.push(`C pageerror: ${e.message}`))
  page.on('console', (m) => { if (m.type() === 'error' && !m.text().includes('401')) errors.push(`C console: ${m.text()}`) })

  await page.goto(`${base}/login`, { waitUntil: 'networkidle' })
  await page.fill('#login-password', '123456')
  await page.click('.submit-btn')
  await page.waitForURL(`${base}/`)
  await page.waitForTimeout(500)
  check('C 端登录进入空态', await page.locator('.chat-empty').isVisible())
  check('空态不展示场景卡', (await page.locator('.scenarios').count()) === 0)
  check('联网搜索入口可见', await page.getByRole('button', { name: '联网搜索' }).isVisible())

  // 新建对话并等待真实 DeepSeek 流式回答完成
  await page.getByRole('button', { name: '新建对话' }).first().click()
  await page.fill('.composer-input', '用一句话解释什么是 SSE。')
  await page.press('.composer-input', 'Enter')
  await page.waitForURL(/\/chat\/\d+$/, { timeout: 15000 })
  await page.waitForFunction(
    () => {
      const assistants = [...document.querySelectorAll('.message.assistant')]
      return assistants.length > 0 && assistants[assistants.length - 1].className.includes('complete')
    },
    null,
    { timeout: 60000 },
  )
  const answer = await page.locator('.dt-prose').last().innerText()
  check('SSE 回答已渲染', answer.length > 5)
  check('模型为 deepseek-flash 展示名', (await page.locator('.assistant-model').last().innerText()).includes('DeepSeek'))
  await page.screenshot({ path: path.join(here, '../../docs/qa/smoke-c-session.png') })

  // 删除测试对话
  await page.locator('.history-item').first().hover()
  await page.locator('.history-item .more').first().click()
  await page.getByRole('menuitem', { name: '删除' }).click()
  await page.locator('.ant-modal-confirm-btns .ant-btn-dangerous').click()
  await page.waitForURL(`${base}/`, { timeout: 10000 })
  check('删除对话回到空态', await page.locator('.chat-empty').isVisible())

  // B 端
  const bctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' })
  const bp = await bctx.newPage()
  bp.on('pageerror', (e) => errors.push(`B pageerror: ${e.message}`))
  bp.on('console', (m) => { if (m.type() === 'error' && !m.text().includes('401')) errors.push(`B console: ${m.text()}`) })
  await bp.goto(`${base}/admin/login`, { waitUntil: 'networkidle' })
  await bp.fill('#admin-password', '123456')
  await bp.click('.submit')
  await bp.waitForURL('**/admin/overview')
  check('B 端登录进入概览', await bp.locator('.overview-page').isVisible())

  await bp.goto(`${base}/admin/providers`, { waitUntil: 'networkidle' })
  await bp.waitForTimeout(500)
  check('仅保留 DeepSeek 一家厂商', (await bp.locator('.provider-card').count()) === 1)
  check('厂商卡片为 DeepSeek', (await bp.locator('.provider-card').first().innerText()).includes('DeepSeek'))
  check('模型路由仅 deepseek-flash', (await bp.locator('.model-table tbody tr').count()) === 1)
  await bp.locator('.provider-card').first().click()
  await bp.waitForSelector('.provider-form')
  check('厂商配置弹窗打开', await bp.getByText('厂商配置', { exact: false }).first().isVisible())
  await bp.locator('.modal-footer .ant-btn').first().click()

  await bp.goto(`${base}/admin/users`, { waitUntil: 'networkidle' })
  await bp.waitForTimeout(500)
  check('用户表仅一个管理员', (await bp.locator('.user-table tbody tr').count()) === 1)
  await bp.locator('.user-table tbody tr').first().click()
  await bp.waitForTimeout(300)
  check('管理员详情为 Wind', (await bp.locator('.user-detail').innerText()).includes('Wind'))
  await bp.locator('.quota-foot button.link').click()
  await bp.waitForSelector('.quota')
  check('额度弹窗打开', await bp.getByText('调整额度', { exact: true }).first().isVisible())
  await bp.locator('.quota .actions .ant-btn').first().click()

  await bp.goto(`${base}/admin/settings`, { waitUntil: 'networkidle' })
  await bp.waitForTimeout(500)
  check('设置页开放注册', (await bp.locator('.status-pill').first().innerText()).includes('开放注册'))
  await bp.locator('.setting-row .ant-switch').click()
  await bp.locator('.ant-modal-confirm-btns .ant-btn-dangerous').click()
  await bp.waitForTimeout(600)
  check('注册开关可关闭', (await bp.locator('.status-pill').first().innerText()).includes('已关闭注册'))
  await bp.locator('.setting-row .ant-switch').click()
  await bp.waitForTimeout(600)
  check('注册开关可恢复', (await bp.locator('.status-pill').first().innerText()).includes('开放注册'))
  await bp.screenshot({ path: path.join(here, '../../docs/qa/smoke-b-settings.png') })

  check('无前端运行时报错', errors.length === 0)
  console.log(JSON.stringify({ ok: true, checked: results.length, errors }, null, 2))
} catch (error) {
  console.error(JSON.stringify({ ok: false, error: error.message, checked: results, errors }, null, 2))
  process.exitCode = 1
} finally {
  await browser.close()
}
