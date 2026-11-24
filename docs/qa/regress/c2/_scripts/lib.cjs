// shared helpers for c2 regression verification (read-only)
const path = require('path')
const deps = path.resolve('/home/xjqxz/Workplace/deeptalk/.browser-deps')
process.env.LD_LIBRARY_PATH = [
  path.join(deps, 'root/usr/lib/x86_64-linux-gnu'),
  path.join(deps, 'root/usr/lib'),
  path.join(deps, 'root/lib/x86_64-linux-gnu'),
  process.env.LD_LIBRARY_PATH || '',
].join(':')
process.env.FONTCONFIG_FILE = path.join(deps, 'fonts.conf')
process.env.FONTCONFIG_PATH = deps

const { chromium } = require('/home/xjqxz/Workplace/deeptalk/ui/node_modules/playwright-core')

const SHOT_DIR = '/home/xjqxz/Workplace/deeptalk/docs/qa/regress/c2'
const H5 = 'http://127.0.0.1:5332/'
const PWA = 'https://deeptalk.peon.top'
const USER = 'xjqxz_2@163.com'
const PASS = 'Ml87148002'

const logs = []
function wire(page, tag) {
  page.on('console', (m) => {
    const t = m.type()
    if (t === 'error' || t === 'warning') logs.push(`[${tag}] console.${t}: ${m.text()}`)
  })
  page.on('pageerror', (e) => logs.push(`[${tag}] pageerror: ${e && e.message ? e.message : e}`))
}

async function launch(opts = {}) {
  return chromium.launch({
    executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`,
    args: ['--no-sandbox'],
    ...opts,
  })
}

module.exports = { chromium, launch, wire, logs, SHOT_DIR, H5, PWA, USER, PASS }
