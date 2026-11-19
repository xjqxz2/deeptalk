import {
  C, text, rect, circle, line, icon, brandMark, pageHeader, appFrame,
  callout, annotationRail, writeSvg, chip, SANS,
} from './req-lib.mjs';

const W = 1440;

function sidebar() {
  const groups = [
    { label: '今天', y: 192, items: [
      ['Go JWT 中间件设计', 210],
      ['React 流式消息渲染', 248],
      ['DeepSeek 风格交互清单', 286],
    ] },
    { label: '7 天内', y: 338, items: [
      ['用户会话隔离方案', 352],
      ['SSE 超时与重连策略', 390],
    ] },
    { label: '30 天内', y: 442, items: [
      ['OpenAI 兼容接口调研', 456],
      ['LangGraph 工作流笔记', 494],
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
    for (const [title, y] of group.items) {
      s += icon('message', 26, y + 10, 15, '#98A2B3', 1.7);
      s += text(50, y + 24, title, { size: 12.2, fill: C.body });
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

function main() {
  let s = rect(0, 0, 1176, 900, { fill: '#FFFFFF' });
  s += rect(0, 0, 1176, 64, { fill: '#FFFFFF' });
  s += line(0, 64, 1176, 64, { stroke: '#F0F2F5' });
  s += icon('panelLeft', 22, 23, 18, C.muted, 1.7);
  s += text(52, 38, '新对话', { size: 13.5, fill: C.ink2, weight: 600 });
  s += circle(112, 34, 3, { fill: C.green });
  s += text(120, 38, '草稿已保存', { size: 10.2, fill: C.subtle });

  s += rect(720, 14, 212, 36, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 10 });
  s += circle(734, 32, 4, { fill: C.primary });
  s += text(746, 36.5, 'DeepSeek-V3.2', { size: 12.3, fill: C.ink2, weight: 600 });
  s += icon('chevronDown', 909, 26, 14, C.subtle, 1.8);
  s += chip(948, 18, 96, '已连接', { h: 28, size: 10.3, fill: C.greenSoft, color: '#027A48', dot: true });
  s += circle(1080, 32, 16, { fill: '#E9EDF5' });
  s += text(1080, 36.5, 'L', { size: 11.5, fill: C.ink2, weight: 650, anchor: 'middle' });
  s += icon('more', 1118, 23, 18, C.subtle, 1.7);

  s += brandMark(560, 216, 56);
  s += text(588, 312, '我是 DeepTalk，很高兴见到你！', { size: 28, weight: 730, anchor: 'middle', spacing: -0.6 });
  s += text(588, 343, '我可以帮你写代码、读文档、做分析，也能陪你一起头脑风暴。', { size: 13.5, fill: C.muted, anchor: 'middle' });

  const cards = [
    { x: 178, icon: 'code', title: '写代码', sub: 'Go / React / SQL' },
    { x: 387, icon: 'file', title: '读文档', sub: '总结 · 提炼 · 对比' },
    { x: 596, icon: 'pen', title: '创意写作', sub: '文案 · 脚本 · 邮件' },
    { x: 805, icon: 'globe', title: '深度研究', sub: '推理 + 联网搜索' },
  ];
  for (const card of cards) {
    s += rect(card.x, 394, 193, 64, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 12 });
    s += rect(card.x + 14, 411, 30, 30, { fill: '#F5F7FF', rx: 9 });
    s += icon(card.icon, card.x + 21, 418, 16, C.primary, 1.7);
    s += text(card.x + 54, 421, card.title, { size: 12.3, fill: C.ink2, weight: 650 });
    s += text(card.x + 54, 439, card.sub, { size: 9.8, fill: C.subtle });
  }

  s += rect(178, 502, 820, 176, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 18, filter: 'softShadow' });
  s += text(202, 542, '给 DeepTalk 发送消息...', { size: 15, fill: C.subtle });
  s += text(202, 566, '支持 Markdown、代码块与表格  ·  Enter 发送 / Shift + Enter 换行', { size: 10.3, fill: C.faint });

  s += icon('paperclip', 202, 629, 18, C.muted, 1.8);
  s += rect(230, 622, 112, 32, { fill: '#F8F9FB', stroke: '#EAECF0', rx: 16 });
  s += icon('spark', 242, 631, 14, C.muted, 1.4);
  s += text(262, 642, '深度思考', { size: 11, fill: C.body, weight: 550 });
  s += rect(352, 622, 108, 32, { fill: '#F8F9FB', stroke: '#EAECF0', rx: 16 });
  s += icon('globe', 364, 631, 14, C.muted, 1.7);
  s += text(384, 642, '联网搜索', { size: 11, fill: C.body, weight: 550 });
  s += text(852, 642, '上下文 32K', { size: 9.8, fill: C.faint });
  s += circle(958, 638, 20, { fill: '#E4E7EC' });
  s += icon('arrowUp', 949, 629, 18, '#98A2B3', 2);

  s += text(588, 706, '内容由 AI 生成，请仔细甄别', { size: 11, fill: C.subtle, anchor: 'middle' });
  return s;
}

const items = [
  { n: 1, title: '侧边栏与历史', lines: ['264px；可折叠为 64px 图标栏。', '按今天 / 7 天 / 30 天分组，支持标题搜索。'] },
  { n: 2, title: '新建对话', lines: ['清空上下文但保留历史；生成中禁止切换。', '会话自动保存，首轮完成后生成短标题。'] },
  { n: 3, title: '账号上下文', lines: ['显示账号与套餐状态；普通用户不接触任何 Key。', '用量、配额与可用模型由后台下发。'] },
  { n: 4, title: '模型选择', lines: ['仅列出管理员启用的模型；用户可切换默认值。', '切换只影响下一条消息，不重写历史。'] },
  { n: 5, title: '空态与快捷提示', lines: ['品牌问候 + 4 个场景卡，点击填入输入框。', '新用户首登展示一次 3 步引导（P1）。'] },
  { n: 6, title: '输入区', lines: ['1–8 行自适应；空内容禁用发送。', 'Enter 发送、Shift+Enter 换行，支持粘贴长文本。'] },
  { n: 7, title: '深度思考 / 联网', lines: ['深度思考透传 reasoning，展示耗时与折叠摘要。', '联网搜索 MVP 仅保留入口，P1 接入搜索服务。'] },
  { n: 8, title: '免责声明与反馈', lines: ['回答与空态均常驻 AI 内容提示。', '错误、限流与配额不足提供明确下一步。'] },
];

const content = `${pageHeader({
  title: 'PC 对话工作台 · 空态',
  subtitle: '首屏、会话历史、模型选择与输入区 · 优先还原 DeepSeek Web 的克制与留白',
  code: 'REQ-01',
})}
${appFrame(`${sidebar()}<g transform="translate(264 0)">${main()}</g>`)}
${callout(1, 288, 184)}
${callout(2, 288, 244)}
${callout(3, 288, 976)}
${callout(4, 1016, 190)}
${callout(5, 1196, 454)}
${callout(6, 1310, 650)}
${callout(7, 1310, 780)}
${callout(8, 1310, 848)}
${annotationRail(items, {
  header: '交互与规则',
  badge: 'MVP / P0',
  footerTitle: '验收标准',
  footerLines: ['1280–1920 宽自适应，1024 折叠侧栏。', '关键操作可键盘完成，对比度 ≥ WCAG AA。'],
  footerNote: 'P1：文件上传、多模态、语音、分享链接、移动端。',
})}`;

writeSvg('docs/requirements/01-pc-chat-empty.svg', content);
