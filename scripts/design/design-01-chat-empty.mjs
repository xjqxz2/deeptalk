import { C, text, rect, circle, icon, brandMark } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, chatSidebar, chatTopbar, writeSvg, g, SHEET } from './design-lib.mjs';

const cards = [
  { icon: 'code', title: '写代码', desc: 'Go · React · SQL' },
  { icon: 'file', title: '读文档', desc: '总结 · 提炼 · 对比' },
  { icon: 'pen', title: '创意写作', desc: '文案 · 脚本 · 邮件' },
  { icon: 'spark', title: '深度研究', desc: '推理 · 联网检索' },
];

function emptyMain() {
  const parts = [];
  const cx = 860;
  parts.push(brandMark(830, 188, 60));
  parts.push(text(cx, 316, '我是 DeepTalk，很高兴见到你！', { size: 30, weight: 730, anchor: 'middle', spacing: -0.5 }));
  parts.push(text(cx, 346, '我可以帮你写代码、读文档、做分析，也能陪你一起头脑风暴。', { size: 13.6, fill: C.muted, anchor: 'middle' }));

  const cardY = 392;
  cards.forEach((c, i) => {
    const x = 484 + i * 192;
    parts.push(rect(x, cardY, 176, 96, { fill: '#FFFFFF', rx: 14, stroke: '#E7EAF0' }));
    parts.push(rect(x + 16, cardY + 16, 36, 36, { fill: C.primarySoft, rx: 11 }));
    parts.push(icon(c.icon, x + 25, cardY + 25, 18, C.primary, 1.8));
    parts.push(text(x + 16, cardY + 70, c.title, { size: 13, weight: 650, fill: C.ink }));
    parts.push(text(x + 16, cardY + 87, c.desc, { size: 10.6, fill: C.subtle }));
  });

  const ix = 484, iy = 544, iw = 752;
  parts.push(rect(ix, iy, iw, 172, { fill: '#FFFFFF', rx: 18, stroke: '#E2E7EF', filter: 'dsgSoft' }));
  parts.push(text(ix + 24, iy + 38, '给 DeepTalk 发送消息…', { size: 13.6, fill: C.subtle }));
  parts.push(text(ix + 24, iy + 62, '支持 Markdown、代码块与表格  ·  Enter 发送 / Shift + Enter 换行', { size: 10.2, fill: C.faint }));
  parts.push(icon('paperclip', ix + 22, iy + 108, 18, C.muted, 1.7));

  parts.push(rect(ix + 54, iy + 100, 104, 32, { fill: C.primarySoft, rx: 16 }));
  parts.push(icon('spark', ix + 64, iy + 109, 14, C.primary, 1.6));
  parts.push(text(ix + 84, iy + 120.5, '深度思考', { size: 11.4, weight: 600, fill: C.primary }));

  parts.push(rect(ix + 168, iy + 100, 104, 32, { fill: '#FFFFFF', rx: 16, stroke: '#E3E7EE' }));
  parts.push(icon('globe', ix + 178, iy + 109, 14, C.muted, 1.6));
  parts.push(text(ix + 198, iy + 120.5, '联网搜索', { size: 11.4, weight: 600, fill: C.ink2 }));

  parts.push(text(ix + iw - 88, iy + 121, '上下文 32K', { size: 10.2, fill: C.subtle, anchor: 'end' }));
  parts.push(circle(ix + iw - 44, iy + 116, 21, { fill: C.primary }));
  parts.push(icon('arrowUp', ix + iw - 53, iy + 107, 18, '#FFFFFF', 2.2));

  parts.push(text(cx, 758, '内容由 AI 生成，请仔细甄别', { size: 10.4, fill: C.faint, anchor: 'middle' }));

  parts.push(pinLabel(1, 268, 96));
  parts.push(pinLabel(2, 264, 209));
  parts.push(pinLabel(3, 1016, 32));
  parts.push(pinLabel(4, 1180, 486));
  parts.push(pinLabel(5, 1204, 700));
  parts.push(pinLabel(6, 268, 806));
  return g(...parts);
}

const inner = g(chatSidebar({ active: 'Go JWT 中间件设计' }), chatTopbar({ title: '新对话' }), emptyMain());

const notes = [
  { n: 1, title: '三段式骨架', lines: ['侧栏 280 固定，内容区弹性；', '顶栏 64，滚动只发生在内容区。'] },
  { n: 2, title: '会话历史', lines: ['按今天 / 7 天 / 30 天分组；', '当前项浅蓝底，hover 出现 ⋯。'] },
  { n: 3, title: '模型选择', lines: ['只列管理员启用的模型；切换', '只影响下一条消息，不追溯历史。'] },
  { n: 4, title: '场景卡', lines: ['176×96 四列等宽；点击写入', '预置 prompt 并聚焦输入框。'] },
  { n: 5, title: '输入区', lines: ['1–8 行自适应；空内容禁用发送；', 'Enter 发送，Shift + Enter 换行。'] },
  { n: 6, title: '用量与账户', lines: ['进度条 32% 对应 1.28M / 4M；', '点击卡片进入用量详情（P1）。'] },
];

const rail = specRail({
  items: notes,
  footerLines: ['1024 以下侧栏收为 64px 图标条', '输入框最大 752，左右留白 40', '焦点环 2px #C9D4FF，动效 160ms'],
});

writeSvg('docs/design/01-chat-empty.svg', sheetHeader({
  title: 'PC 对话工作台 · 空态',
  sub: '首屏即工作台：历史、模型、场景卡与输入区一次到位',
  code: 'DSG-01 / CHAT-EMPTY',
}) + frame(32, 142, 1440, 900, 'screen01', inner) + rail, 1800, 1080);
