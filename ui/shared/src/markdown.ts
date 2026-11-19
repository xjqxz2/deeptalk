/**
 * Markdown 渲染：GFM 基础语法 + 代码高亮 + 代码块复制。
 * 安全边界：markdown-it 关闭原生 HTML，DOMPurify 再做一次白名单清洗，
 * 保证模型输出（不可信内容）无法注入脚本。
 */
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import 'highlight.js/styles/github-dark.css'

/** markdown-it 实例；html:false 从源头阻止原始 HTML。 */
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false,
})

/** 代码块首行若像文件名，就提取到头部展示，正文不再重复。 */
const FILE_PATTERN = /^[\w./-]+\.(go|ts|tsx|js|jsx|py|json|sh|sql|vue|css|html|md|yaml|yml)$/

/** 语言展示名映射，避免头部出现小写语言名。 */
const LANGUAGE_LABELS: Record<string, string> = {
  go: 'Go',
  ts: 'TypeScript',
  tsx: 'TSX',
  js: 'JavaScript',
  jsx: 'JSX',
  py: 'Python',
  json: 'JSON',
  sh: 'Shell',
  bash: 'Shell',
  sql: 'SQL',
  vue: 'Vue',
  css: 'CSS',
  html: 'HTML',
  yaml: 'YAML',
  yml: 'YAML',
  text: 'Text',
}

/** 替换默认 fence 渲染器，输出带文件名、语言与复制按钮的代码块。 */
md.renderer.rules.fence = (tokens, idx) => {
  const token = tokens[idx]
  const info = (token.info || '').trim()
  const language = (info.split(/\s+/)[0] || 'text').toLowerCase()
  let code = token.content.replace(/\n$/, '')
  let filename = ''
  const firstLine = code.split('\n')[0]?.trim() ?? ''
  if (FILE_PATTERN.test(firstLine)) {
    filename = firstLine
    code = code.split('\n').slice(1).join('\n').replace(/^\n+/, '')
  }
  let highlighted: string
  if (language !== 'text' && hljs.getLanguage(language)) {
    highlighted = hljs.highlight(code, { language, ignoreIllegals: true }).value
  } else {
    highlighted = md.utils.escapeHtml(code)
  }
  const label = filename || LANGUAGE_LABELS[language] || language
  const langBadge = filename ? LANGUAGE_LABELS[language] || language : ''
  return [
    '<div class="dt-code">',
    '<div class="dt-code-head">',
    `<span class="dt-code-label">${md.utils.escapeHtml(label)}</span>`,
    langBadge ? `<span class="dt-code-lang">${md.utils.escapeHtml(langBadge)}</span>` : '<span></span>',
    '<button class="dt-code-copy" type="button" aria-label="复制代码">复制</button>',
    '</div>',
    `<pre class="dt-code-pre"><code class="hljs language-${md.utils.escapeHtml(language)}">${highlighted}</code></pre>`,
    '</div>',
  ].join('')
}

/** 外链统一新窗口打开，并补 rel 防止 opener 攻击。 */
md.renderer.rules.link_open = (tokens, idx, options, _env, self) => {
  const token = tokens[idx]
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return self.renderToken(tokens, idx, options)
}

/** 渲染 Markdown 为可安全插入的 HTML。 */
export function renderMarkdown(text: string): string {
  if (!text) return ''
  const html = md.render(text)
  return DOMPurify.sanitize(html, {
    ADD_ATTR: ['target', 'rel'],
    FORBID_TAGS: ['style', 'form', 'input', 'iframe', 'object', 'embed'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick'],
  })
}

/**
 * 为 Markdown 容器绑定代码复制行为。
 * 使用事件委托，容器内容被重新渲染后仍然有效；重复调用不会重复绑定。
 */
export function enhanceMarkdown(root: HTMLElement | null | undefined): void {
  if (!root || root.dataset.copyBound === '1') return
  root.dataset.copyBound = '1'
  root.addEventListener('click', (event) => {
    const target = event.target as HTMLElement | null
    const button = target?.closest<HTMLButtonElement>('.dt-code-copy')
    if (!button) return
    const code = button.closest('.dt-code')?.querySelector('code')?.textContent ?? ''
    void copyText(code).then((ok) => {
      const original = button.textContent
      button.textContent = ok ? '已复制' : '复制失败'
      window.setTimeout(() => {
        button.textContent = original
      }, 1400)
    })
  })
}

/** 复制文本，优先使用 Clipboard API，失败时回退到临时 textarea。 */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // 非安全上下文或权限被拒绝时继续走降级方案
  }
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}
