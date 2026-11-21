/**
 * A2UI（v0.9）消息 → 小程序可直接渲染的结构。
 *
 * 官方渲染器 @a2ui/lit 依赖 Web Components 与 DOM，微信小程序没有这些能力，
 * 但 A2UI 本身是声明式 JSON（createSurface / updateComponents / updateDataModel），
 * 所以可以自己实现渲染层：这里把消息折叠成「区块 → 条目」两层结构，
 * 再用 uni 的 view/text/button 渲染。
 *
 * 只覆盖基础 catalog 里对话真正会用到的组件：Text / Column / Row / Card / List /
 * Divider / Image / Button。遇到其它类型会降级成一个提示，不会白屏。
 */

/** 一次最多渲染的区块数，防止异常数据把页面撑爆。 */
const MAX_SECTIONS = 40
/** 一个区块里最多渲染的条目数。 */
const MAX_ITEMS = 60

/** 把 A2UI 消息数组折叠成渲染模型。 */
export function buildA2uiModel(messages = []) {
  const nodes = {}
  const data = {}
  let rootId = ''

  for (const message of messages) {
    if (!message || typeof message !== 'object') continue
    if (message.createSurface) {
      const surface = message.createSurface
      rootId = surface.rootId || rootId || ''
    }
    if (message.updateComponents) {
      const list = message.updateComponents.components || []
      for (const component of list) {
        if (!component || !component.id) continue
        nodes[component.id] = component
        if (component.id === 'root') rootId = 'root'
      }
    }
    if (message.updateDataModel) {
      const update = message.updateDataModel
      const path = update.path || '/'
      if (path === '/' || path === '') {
        Object.assign(data, update.value || {})
      } else {
        // 只支持一层路径，够用且不会写出意外结构
        data[path.replace(/^\//, '')] = update.value
      }
    }
  }

  return { rootId: rootId || 'root', nodes, data }
}

/** 解析文本型属性：字符串 / {path} / {literal}。 */
export function resolveValue(value, data) {
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (!value || typeof value !== 'object') return ''
  if (typeof value.literal === 'string') return value.literal
  if (typeof value.literal === 'number') return String(value.literal)
  if (typeof value.path === 'string') {
    const key = value.path.replace(/^\//, '')
    const found = data[key]
    if (found == null) return ''
    if (typeof found === 'object') return JSON.stringify(found)
    return String(found)
  }
  return ''
}

/** 取子节点。Card 用 child，其余用 children。 */
function childIds(node) {
  if (!node) return []
  if (Array.isArray(node.children)) return node.children
  if (typeof node.child === 'string') return [node.child]
  if (node.child && typeof node.child === 'string') return [node.child]
  return []
}

/** 把节点转换成渲染条目；容器会展开成对应的条目。 */
function toItem(node, model) {
  if (!node || !node.component) return null
  const { data } = model
  switch (node.component) {
    case 'Text': {
      const text = resolveValue(node.text, data)
      if (!text) return null
      return { id: node.id, kind: 'text', text, variant: node.variant || 'body', weight: node.weight || '' }
    }
    case 'Divider':
      return { id: node.id, kind: 'divider' }
    case 'Image': {
      const url = resolveValue(node.url, data)
      if (!url) return null
      return { id: node.id, kind: 'image', url, alt: resolveValue(node.alt, data) }
    }
    case 'Icon':
      return { id: node.id, kind: 'icon', name: resolveValue(node.name, data) }
    case 'Button': {
      const ids = childIds(node)
      const label = ids.map((id) => resolveValue(model.nodes[id]?.text, data)).filter(Boolean).join('')
      return { id: node.id, kind: 'button', label: label || '按钮', action: node.action || null, variant: node.variant || 'default' }
    }
    case 'Row':
    case 'List': {
      const items = childIds(node).map((id) => toItem(model.nodes[id], model)).filter(Boolean)
      if (!items.length) return null
      return { id: node.id, kind: node.component === 'Row' ? 'row' : 'list', items: items.slice(0, MAX_ITEMS) }
    }
    case 'Column': {
      const items = childIds(node).map((id) => toItem(model.nodes[id], model)).filter(Boolean)
      if (!items.length) return null
      return { id: node.id, kind: 'column', items: items.slice(0, MAX_ITEMS) }
    }
    case 'Card': {
      const items = childIds(node).map((id) => toItem(model.nodes[id], model)).filter(Boolean)
      return { id: node.id, kind: 'card', items: items.slice(0, MAX_ITEMS) }
    }
    default:
      return { id: node.id, kind: 'unsupported', name: node.component }
  }
}

/**
 * 折叠成区块列表。
 * 顶层 Column 的每个子节点是一个区块；Card/Row/List 内部的容器再展开一层。
 */
export function buildSections(messages = []) {
  const model = buildA2uiModel(messages)
  const root = model.nodes[model.rootId]
  if (!root) return []

  const sections = []
  const pushSection = (node) => {
    if (sections.length >= MAX_SECTIONS) return
    if (node.component === 'Card') {
      const items = childIds(node)
        .map((id) => toItem(model.nodes[id], model))
        .filter(Boolean)
        // Card 里常见 Column 包一层，这里拍平，避免再嵌一层
        .flatMap((item) => (item.kind === 'column' ? item.items : [item]))
      sections.push({ id: node.id, kind: 'card', items: items.slice(0, MAX_ITEMS) })
      return
    }
    if (node.component === 'Column') {
      const items = childIds(node).map((id) => toItem(model.nodes[id], model)).filter(Boolean)
      sections.push({ id: node.id, kind: 'plain', items: items.slice(0, MAX_ITEMS) })
      return
    }
    const item = toItem(node, model)
    if (item) sections.push({ id: node.id, kind: 'plain', items: [item] })
  }

  const rootChildren = childIds(root)
  if (root.component === 'Column' && rootChildren.length) {
    for (const id of rootChildren) {
      const node = model.nodes[id]
      if (node) pushSection(node)
    }
  } else {
    pushSection(root)
  }

  return sections
}
