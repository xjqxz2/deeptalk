/**
 * Markdown 解析回归测试：
 * 重点是「任何输入都必须在有限时间内结束」——之前四反引号代码块会让解析器
 * 死循环，进而把浏览器标签页拖到 Out of Memory。
 */
import { readFileSync, readdirSync } from 'node:fs'
import { parseMarkdown, parseInline } from '../src/utils/markdown.js'

let failed = 0
const check = (name, condition, extra = '') => {
  if (condition) {
    console.log('✓', name)
  } else {
    console.error('✗', name, extra)
    failed += 1
  }
}

// 1. 四反引号围栏（模型常见的嵌套代码包装）
const four = parseMarkdown('前言\n\n````markdown\n# 标题\n````\n结尾')
check('四反引号能识别为代码块', four.some((b) => b.type === 'code' && b.language === 'markdown' && b.code === '# 标题'), JSON.stringify(four))
check('四反引号后的正文保留', four.some((b) => b.type === 'paragraph' && b.spans[0].text === '结尾'))

// 2. 常规边界
check('未闭合围栏不挂起', parseMarkdown('```js\nconst a = 1\n').length === 1)
check('空围栏不挂起', parseMarkdown('```\n```').length === 1)
check('空输入返回空数组', parseMarkdown('').length === 0)
check('纯空行返回空数组', parseMarkdown('\n\n\n').length === 0)
check('表格行不挂起', parseMarkdown('| a | b |\n|---|---|\n| 1 | 2 |').length === 1)
check('列表正常', parseMarkdown('- a\n- b').length === 1)
check('引用正常', parseMarkdown('> 引用').length === 1)
check('标题正常', parseMarkdown('## 标题').length === 1)

// 3. 行内解析
check('加粗', parseInline('**粗**')[0].bold === true)
check('行内代码', parseInline('`code`')[0].code === true)
check('引用标号', parseInline('参考[1]')[1].cite === true)
check('链接', parseInline('[名字](https://a.b)')[0].link === 'https://a.b')

// 4. 真实语料：用抓下来的线上会话跑一遍，确保不挂起
const files = process.env.CONV_DIR
  ? readdirSync(process.env.CONV_DIR).filter((f) => f.startsWith('c-') && f.endsWith('.json'))
  : []
let messages = 0
for (const file of files) {
  const data = JSON.parse(readFileSync(`${process.env.CONV_DIR}/${file}`, 'utf8')).data
  for (const message of data.messages || []) {
    if (!message.content) continue
    const started = Date.now()
    parseMarkdown(message.content)
    const cost = Date.now() - started
    messages += 1
    if (cost > 500) {
      console.error('✗ 解析过慢', file, message.id, cost + 'ms')
      failed += 1
    }
  }
}
if (files.length) console.log(`✓ 真实语料 ${messages} 条消息全部解析完成`)

if (failed) {
  console.error(`\n${failed} 项失败`)
  process.exit(1)
}
console.log('\n全部通过')
