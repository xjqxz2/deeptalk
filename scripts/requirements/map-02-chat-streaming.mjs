import {
  C, text, tspan, rect, circle, line, path, icon, brandMark, pageHeader, appFrame,
  callout, annotationRail, writeSvg, chip, MONO,
} from './req-lib.mjs';

function sidebar() {
  const groups = [
    { label: '今天', y: 192, items: [
      ['Go SSE 流式转发', 210, true],
      ['React 流式消息渲染', 248, false],
      ['用户会话隔离方案', 286, false],
    ] },
    { label: '7 天内', y: 338, items: [
      ['DeepSeek 风格交互清单', 352, false],
      ['SSE 超时与重连策略', 390, false],
    ] },
    { label: '30 天内', y: 442, items: [
      ['OpenAI 兼容接口调研', 456, false],
    ] },
  ];

  let s = rect(0, 0, 264, 900, { fill: '#F8F9FB' });
  s += line(264, 0, 264, 900, { stroke: '#E8EBF0' });
  s += brandMark(20, 20, 30);
  s += text(60, 40, 'DeepTalk', { size: 18, weight: 730, spacing: -0.3 });
  s += chip(147, 25, 36, 'MVP', { h: 18, size: 8.5, fill: C.primarySoft, color: C.primary });
  s += icon('panelLeft', 226, 25, 18, C.muted, 1.7);

  s += rect(16, 78, 232, 42, { fill: C.primarySoft, rx: 11 });
  s += icon('plus', 30, 90, 18, C.primary, 2);
  s += text(56, 104, '新建对话', { size: 13.5, weight: 650, fill: C.primary2 });
  s += text(214, 104, 'Ctrl K', { size: 9.5, fill: '#7C8DB5', anchor: 'end' });

  s += rect(16, 134, 232, 38, { fill: '#EFF1F5', rx: 10 });
  s += icon('search', 30, 144, 16, C.subtle, 1.8);
  s += text(56, 158, '搜索对话', { size: 12.5, fill: C.subtle });

  for (const group of groups) {
    s += text(22, group.y, group.label, { size: 10.5, fill: C.subtle, weight: 600, spacing: 0.5 });
    for (const [title, y, active] of group.items) {
      if (active) {
        s += rect(12, y, 240, 36, { fill: '#E9EEFF', rx: 9 });
        s += rect(12, y + 8, 3, 20, { fill: C.primary, rx: 1.5 });
      }
      s += icon('message', 26, y + 10, 15, active ? C.primary : '#98A2B3', 1.7);
      s += text(50, y + 24, title, { size: 12.2, fill: active ? C.primary2 : C.body, weight: active ? 600 : 400 });
    }
  }

  s += rect(16, 724, 232, 74, { fill: '#FFFFFF', stroke: '#E7EAF0', rx: 12 });
  s += icon('spark', 28, 736, 14, C.primary, 1.5);
  s += text(48, 748, '本月用量', { size: 10.8, fill: C.muted, weight: 550 });
  s += text(232, 748, '32%', { size: 10.8, fill: C.ink2, weight: 650, anchor: 'end' });
  s += rect(28, 760, 208, 6, { fill: '#EAECF0', rx: 3 });
  s += rect(28, 760, 67, 6, { fill: C.primary, rx: 3 });
  s += text(28, 786, '1.28M / 4M tokens', { size: 9.8, fill: C.subtle });
  s += text(232, 786, '重置 12 天', { size: 9.5, fill: C.subtle, anchor: 'end' });

  s += circle(32, 842, 16, { fill: '#DFE6FF' });
  s += text(32, 847, 'L', { size: 12.5, fill: C.primary2, weight: 700, anchor: 'middle' });
  s += text(58, 837, '林默', { size: 12.2, fill: C.ink2, weight: 600 });
  s += circle(60, 852, 2.8, { fill: C.green });
  s += text(68, 855, '管理员 · 免费版', { size: 9.8, fill: C.subtle });
  s += icon('settings', 220, 834, 16, C.subtle, 1.7);
  return s;
}

function topbar() {
  let s = rect(0, 0, 1176, 64, { fill: '#FFFFFF' });
  s += line(0, 64, 1176, 64, { stroke: '#F0F2F5' });
  s += icon('panelLeft', 22, 23, 18, C.muted, 1.7);
  s += text(52, 38, 'Go SSE 流式转发', { size: 13.5, fill: C.ink2, weight: 650 });
  s += text(176, 38, '已自动命名', { size: 9.8, fill: C.subtle });
  s += rect(772, 14, 212, 36, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 10 });
  s += circle(786, 32, 4, { fill: C.primary });
  s += text(798, 36.5, 'DeepSeek-V3.2', { size: 12.3, fill: C.ink2, weight: 600 });
  s += icon('chevronDown', 961, 26, 14, C.subtle, 1.8);
  s += icon('share', 1008, 24, 17, C.subtle, 1.7);
  s += icon('more', 1048, 23, 18, C.subtle, 1.7);
  s += circle(1108, 32, 16, { fill: '#E9EDF5' });
  s += text(1108, 36.5, 'L', { size: 11.5, fill: C.ink2, weight: 650, anchor: 'middle' });
  return s;
}

const GO_KEYWORDS = new Set(['func', 'defer', 'for', 'if', 'return', 'break', 'range', 'var', 'go']);
const GO_TYPES = new Set(['string', 'byte', 'int', 'error', 'bool', 'http', 'bufio', 'gin', 'context']);

function goLine(lineText, x, y) {
  const tokens = lineText.match(/("(?:[^"\\]|\\.)*"|\b(?:func|defer|for|if|return|break|range|var|go|string|byte|int|error|bool|http|bufio|gin|context)\b|\b[A-Za-z_]\w*(?=\()|[{}()[\],.;:=<>+\-*/]|\s+|.)/g) || [];
  let out = '';
  for (const tok of tokens) {
    let color = C.codeText;
    if (/^"/.test(tok)) color = '#C3E88D';
    else if (GO_KEYWORDS.has(tok)) color = '#C792EA';
    else if (GO_TYPES.has(tok)) color = '#FFCB6B';
    else if (/^[A-Za-z_]\w*$/.test(tok)) color = '#82AAFF';
    else if (/^[{}()[\],.;:=<>+\-*/]$/.test(tok)) color = '#A6ACCD';
    out += tspan(tok, { fill: color, family: MONO });
  }
  return text(x, y, out, { size: 10.6, family: MONO, raw: true });
}

function codeBlock() {
  const code = [
    'func proxySSE(c *gin.Context, up *http.Response) {',
    '  defer up.Body.Close()',
    '  c.Header("Content-Type", "text/event-stream")',
    '  c.Header("X-Accel-Buffering", "no")',
    '  flusher := c.Writer.(http.Flusher)',
    '  reader := bufio.NewReader(up.Body)',
    '  for {',
    '    line, err := reader.ReadBytes(\'\\n\')',
    '    if len(line) > 0 { c.Writer.Write(line); flusher.Flush() }',
    '    if err != nil { break }',
    '  }',
    '}',
  ];
  let s = rect(208, 400, 760, 252, { fill: C.codeBg, rx: 12 });
  s += path('M208 412 Q208 400 220 400 H956 Q968 400 968 412 V438 H208 Z', { fill: C.codeHeader });
  s += circle(226, 418, 4, { fill: '#FF5F57' });
  s += circle(240, 418, 4, { fill: '#FEBC2E' });
  s += circle(254, 418, 4, { fill: '#28C840' });
  s += text(276, 422, 'sse_proxy.go', { size: 9.8, fill: '#98A2B3' });
  s += text(368, 422, 'Go', { size: 9.6, fill: '#7F8C98' });
  s += icon('copy', 928, 411, 15, '#98A2B3', 1.6);
  s += text(948, 422, '复制', { size: 9.5, fill: '#98A2B3' });
  let y = 458;
  for (const lineText of code) {
    s += goLine(lineText, 228, y);
    y += 16;
  }
  return s;
}

function main() {
  let s = rect(0, 0, 1176, 900, { fill: '#FFFFFF' });
  s += topbar();

  // user message
  s += rect(420, 88, 540, 76, { fill: '#F2F4F7', rx: 16 });
  s += text(442, 118, '帮我用 Go 实现一个 SSE 流式转发，要支持客户端断开', { size: 13.2, fill: C.ink2 });
  s += text(442, 141, '和上游超时，代码尽量精简。', { size: 13.2, fill: C.ink2 });
  s += circle(974, 102, 15, { fill: '#E9EDF5' });
  s += text(974, 106, '你', { size: 10.5, fill: C.muted, anchor: 'middle' });

  // assistant identity
  s += brandMark(208, 192, 28);
  s += text(246, 210, 'DeepTalk', { size: 13, weight: 700 });
  s += chip(314, 195, 43, 'V3.2', { h: 20, size: 8.8, fill: '#F5F7FF', color: C.primary2 });
  s += text(968, 209, '刚刚', { size: 9.5, fill: C.faint, anchor: 'end' });

  // reasoning collapsed row + expanded body
  s += rect(208, 226, 760, 38, { fill: '#F7F8FA', rx: 10 });
  s += icon('spark', 224, 237, 16, C.primary, 1.4);
  s += text(248, 250, '已深度思考（用时 6 秒）', { size: 11.4, fill: C.muted, weight: 550 });
  s += icon('chevronUp', 938, 237, 16, C.subtle, 1.8);
  s += line(222, 276, 222, 326, { stroke: '#D9E1FF', sw: 2 });
  s += text(238, 292, '先确认 SSE 的取消语义与 flush 时机，再检查上游响应头与缓冲。', { size: 10.6, fill: C.muted });
  s += text(238, 311, '方案：独立 goroutine 读上游 → 事件通道 → 客户端断开时 cancel。', { size: 10.6, fill: C.muted });

  // answer
  const answerLine1 = `${tspan('可以。核心是把上游的 ')}${tspan('text/event-stream', { fill: C.primary2, family: MONO })}${tspan(' 直接透传，并用 ')}${tspan('context', { fill: C.primary2, family: MONO })}${tspan(' 同时监听')}`;
  s += text(208, 356, answerLine1, { size: 13.2, fill: C.ink2, raw: true });
  s += text(208, 379, '客户端断开与总超时。下面是精简实现：', { size: 13.2, fill: C.ink2 });

  s += codeBlock();
  s += text(208, 674, '注意：生产环境还需限制单次响应大小，并透传上游的错误事件。', { size: 11.2, fill: C.muted });

  // message actions
  s += icon('copy', 208, 694, 15, C.subtle, 1.7);
  s += text(228, 706, '复制', { size: 10.3, fill: C.subtle });
  s += icon('refresh', 268, 694, 15, C.subtle, 1.7);
  s += text(288, 706, '重新生成', { size: 10.3, fill: C.subtle });
  s += icon('thumbUp', 350, 694, 15, C.subtle, 1.7);
  s += icon('thumbDown', 380, 694, 15, C.subtle, 1.7);
  s += text(968, 706, '内容由 AI 生成，请仔细甄别', { size: 9.5, fill: C.faint, anchor: 'end' });

  // sticky composer
  s += rect(0, 704, 1176, 196, { fill: 'url(#fadeWhite)' });
  s += rect(208, 726, 760, 126, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 18, filter: 'softShadow' });
  s += text(232, 760, '继续追问...', { size: 13.2, fill: C.subtle });
  s += circle(235, 782, 3, { fill: C.primary });
  s += text(245, 786, '正在生成回答…', { size: 10.3, fill: C.primary2 });
  s += icon('paperclip', 232, 814, 17, C.muted, 1.8);
  s += rect(260, 806, 108, 30, { fill: '#F8F9FB', stroke: '#EAECF0', rx: 15 });
  s += icon('spark', 272, 814, 14, C.muted, 1.4);
  s += text(292, 825, '深度思考', { size: 10.6, fill: C.body });
  s += rect(378, 806, 104, 30, { fill: '#F8F9FB', stroke: '#EAECF0', rx: 15 });
  s += icon('globe', 390, 814, 14, C.muted, 1.7);
  s += text(410, 825, '联网搜索', { size: 10.6, fill: C.body });
  s += circle(936, 822, 20, { fill: C.primary });
  s += icon('stop', 927, 813, 18, '#FFFFFF', 1.8);
  s += text(588, 878, '内容由 AI 生成，请仔细甄别', { size: 10.3, fill: C.subtle, anchor: 'middle' });
  return s;
}

const items = [
  { n: 1, title: '流式输出', lines: ['SSE 逐 token 渲染；首字 < 2s（上游允许时）。', '自动跟随；用户上滚后暂停并显示“回到底部”。'] },
  { n: 2, title: '深度思考', lines: ['独立折叠区，展示耗时与推理摘要。', '上游无 reasoning 时自动隐藏，不占位。'] },
  { n: 3, title: 'Markdown / 代码', lines: ['GFM、表格、公式与代码高亮；代码块可复制。', '超长代码横向滚动，不撑破消息宽度。'] },
  { n: 4, title: '消息操作', lines: ['复制、重新生成、赞/踩；失败可重试。', '重新生成覆盖当前回答，保留原提示词。'] },
  { n: 5, title: '停止生成', lines: ['发送键切换为停止；中止上游请求。', '保留已输出内容并标记“已停止”。'] },
  { n: 6, title: '异常与重试', lines: ['429 / 超时 / 5xx：保留用户消息，展示可读错误。', '重试不重复插入用户消息，不丢上下文。'] },
  { n: 7, title: '上下文管理', lines: ['默认最近 20 条 / 32k tokens；超限截断最早消息。', '每轮记录 token 用量，配额不足提前拦截。'] },
  { n: 8, title: '自动标题', lines: ['首轮完成后异步生成 ≤16 字标题。', '支持手动重命名，搜索索引即时更新。'] },
];

const content = `${pageHeader({
  title: '对话进行态 · 流式回答',
  subtitle: '流式输出、深度思考、Markdown 代码块与消息操作 · 还原 DeepSeek 的阅读节奏',
  code: 'REQ-02',
})}
${appFrame(`${sidebar()}<g transform="translate(264 0)">${main()}</g>`)}
${callout(1, 1276, 250)}
${callout(2, 1276, 400)}
${callout(3, 1276, 620)}
${callout(4, 1276, 846)}
${callout(5, 1276, 968)}
${callout(7, 1276, 720)}
${annotationRail(items, {
  header: '交互与规则',
  badge: 'MVP / P0',
  footerTitle: '验收标准',
  footerLines: ['流式渲染不卡顿；复制内容完整；刷新可恢复会话。', '错误态可重试，且不重复插入用户消息。'],
  footerNote: 'P1：编辑消息、分支对话、文件与多模态。',
})}`;

writeSvg('docs/requirements/02-chat-streaming.svg', content);
