/**
 * 轻量 Markdown 解析：把模型输出转成 nvue 能渲染的结构。
 *
 * nvue 不支持 v-html，也没有 DOM，所以不能直接用 markdown-it + 富文本。
 * 这里只实现对话里真正会出现的语法：标题、段落、列表、引用、代码块，
 * 以及行内的加粗 / 斜体 / 行内代码 / 链接 / 引用标号 [1]。
 *
 * 注意：解析循环必须保证每次迭代都前进，否则遇到非预期输入（例如模型用
 * 四个反引号包裹代码）会退化成死循环并把浏览器内存吃光。
 */

/** 行内解析：返回 span 数组，供 <text> 嵌套渲染。 */
export function parseInline(text) {
  const spans = []
  const pattern = /(\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\[[^\]]+\]\([^)]+\)|\[\d{1,2}\])/g
  let last = 0
  let match
  while ((match = pattern.exec(text)) !== null) {
    if (match.index > last) spans.push({ text: text.slice(last, match.index) })
    const token = match[0]
    if (token.startsWith('**') || token.startsWith('__')) {
      spans.push({ text: token.slice(2, -2), bold: true })
    } else if (token.startsWith('`')) {
      spans.push({ text: token.slice(1, -1), code: true })
    } else if (token.startsWith('[') && token.includes('](')) {
      const label = token.slice(1, token.indexOf(']'))
      const url = token.slice(token.indexOf('](') + 2, -1)
      spans.push({ text: label, link: url })
    } else {
      spans.push({ text: token, cite: true })
    }
    last = match.index + token.length
  }
  if (last < text.length) spans.push({ text: text.slice(last) })
  return spans.length ? spans : [{ text }]
}

/** 是否是代码块围栏行，返回 ``` 的个数（不足三个为 0）。 */
function fenceLength(line) {
  const match = /^(`{3,})[^`]*$/.exec(line)
  return match ? match[1].length : 0
}

/**
 * 解析整段 Markdown。
 * @param {string} source 模型输出的原文
 * @returns {Array<object>} 可被 message-item 渲染的块
 */
export function parseMarkdown(source) {
  const text = String(source || '').replace(/\r\n/g, '\n')
  const blocks = []
  const lines = text.split('\n')
  let index = 0

  while (index < lines.length) {
    // 兜底：任何分支如果没有消费行，就强制前进一行，避免死循环
    const startIndex = index
    const line = lines[index]

    const fence = fenceLength(line)
    if (fence > 0) {
      const language = line.slice(fence).trim() || 'text'
      const content = []
      index += 1
      while (index < lines.length) {
        const closing = fenceLength(lines[index])
        // 同长度或更长的围栏才算闭合
        if (closing >= fence) break
        content.push(lines[index])
        index += 1
      }
      if (index < lines.length) index += 1
      blocks.push({ type: 'code', language, code: content.join('\n') })
      continue
    }

    // 空行
    if (!line.trim()) {
      index += 1
      continue
    }

    // 标题
    const heading = /^(#{1,6})\s+(.*)$/.exec(line)
    if (heading) {
      blocks.push({ type: 'heading', level: heading[1].length, spans: parseInline(heading[2]) })
      index += 1
      continue
    }

    // 引用
    if (/^>\s?/.test(line)) {
      const content = []
      while (index < lines.length && /^>\s?/.test(lines[index])) {
        content.push(lines[index].replace(/^>\s?/, ''))
        index += 1
      }
      blocks.push({ type: 'quote', spans: parseInline(content.join(' ')) })
      continue
    }

    // 有序 / 无序列表
    const listMatch = /^\s*([-*+]|\d+\.)\s+/.exec(line)
    if (listMatch) {
      const ordered = /\d/.test(listMatch[1])
      const items = []
      while (index < lines.length) {
        const item = /^\s*([-*+]|\d+\.)\s+(.*)$/.exec(lines[index])
        if (!item) break
        items.push({ spans: parseInline(item[2]) })
        index += 1
      }
      blocks.push({ type: 'list', ordered, items })
      continue
    }

    // 表格：按行读成纯文本块，避免竖线把正文撑坏
    if (/^\s*\|.*\|\s*$/.test(line)) {
      const rows = []
      while (index < lines.length && /^\s*\|.*\|\s*$/.test(lines[index])) {
        rows.push(lines[index].trim())
        index += 1
      }
      blocks.push({ type: 'paragraph', spans: parseInline(rows.join('\n')) })
      continue
    }

    // 段落：连续非空行合并
    const paragraph = []
    while (index < lines.length) {
      const current = lines[index]
      if (
        !current.trim() ||
        fenceLength(current) > 0 ||
        /^#{1,6}\s/.test(current) ||
        /^>\s?/.test(current) ||
        /^\s*([-*+]|\d+\.)\s+/.test(current) ||
        /^\s*\|.*\|\s*$/.test(current)
      ) {
        break
      }
      paragraph.push(current)
      index += 1
    }
    if (paragraph.length) {
      blocks.push({ type: 'paragraph', spans: parseInline(paragraph.join('\n')) })
    }

    // 极端输入兜底：这一段什么都没吃掉就前进一行
    if (index === startIndex) index += 1
  }

  return blocks
}
