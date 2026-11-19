import { C, MONO, SANS, text, rect, circle, line, icon, brandMark, esc } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, chatSidebar, chatTopbar, writeSvg, g } from './design-lib.mjs';

const L = 508;
const W = 704;
const R = L + W;

const TOK = {
  kw: '#7AA2F7',
  type: '#BB9AF7',
  str: '#9ECE6A',
  cm: '#6B7A90',
  num: '#FF9E64',
  fn: '#7DCFFF',
  plain: '#D6DEEB',
  punct: '#A9B4C6',
};

function codeLine(x, y, segs, size = 11) {
  const list = Array.isArray(segs[0][0]) ? segs[0] : segs;
  const inner = list.map(([s, c]) => '<tspan fill="' + c + '">' + esc(s) + '</tspan>').join('');
  return '<text x="' + x + '" y="' + y + '" font-family="' + MONO + '" font-size="' + size + '" xml:space="preserve">' + inner + '</text>';
}

const CODE = [
  [[['func ', TOK.kw], ['(p *Proxy) ', TOK.plain], ['Stream', TOK.fn], ['(w http.ResponseWriter, r *http.Request) {', TOK.plain]]],
  [[['    cfg := p.cfg.SSE  ', TOK.plain], ['// 超时 / 退避 / 重试统一配置', TOK.cm]]],
  [[['    ctx, cancel := ', TOK.plain], ['context.', TOK.type], ['WithCancel', TOK.fn], ['(r.Context())', TOK.plain]]],
  [[['    ', TOK.plain], ['defer ', TOK.kw], ['cancel()', TOK.plain]]],
  [[['    up, err := p.', TOK.plain], ['connect', TOK.fn], ['(ctx, cfg)', TOK.plain]]],
  [[['    ', TOK.plain], ['if ', TOK.kw], ['err != ', TOK.plain], ['nil ', TOK.kw], ['{ p.', TOK.plain], ['fail', TOK.fn], ['(w, err); ', TOK.plain], ['return ', TOK.kw], ['}', TOK.plain]]],
  [[['    ', TOK.plain], ['defer ', TOK.kw], ['up.', TOK.plain], ['Close', TOK.fn], ['()', TOK.plain]]],
  [[['    w.Header().', TOK.plain], ['Set', TOK.fn], ['(', TOK.plain], ['"Content-Type"', TOK.str], [', ', TOK.plain], ['"text/event-stream"', TOK.str], [')', TOK.plain]]],
  [[['    ', TOK.plain], ['return ', TOK.kw], ['relay', TOK.fn], ['(ctx, w, up, cfg.Retry)', TOK.plain]]],
  [[['}', TOK.plain]]],
];

function thread() {
  const parts = [];
  parts.push(rect(888, 84, 324, 78, { fill: '#F1F3F7', rx: 16 }));
  parts.push(text(908, 116, '帮我把这段 SSE 代理的超时与重连逻辑', { size: 13.2, fill: C.ink2 }));
  parts.push(text(908, 138, '改成可配置的，并补上单元测试。', { size: 13.2, fill: C.ink2 }));
  parts.push(text(R, 180, '14:32', { size: 9.8, fill: C.faint, anchor: 'end', family: MONO }));

  parts.push(brandMark(L, 202, 26));
  parts.push(text(L + 34, 220, 'DeepTalk', { size: 12.2, weight: 650 }));
  parts.push(text(L + 112, 220, 'DeepSeek-V3.2 · 深度思考', { size: 10.2, fill: C.subtle }));

  parts.push(rect(L, 238, W, 42, { fill: '#F7F8FA', rx: 10, stroke: '#ECEEF2' }));
  parts.push(icon('spark', L + 14, 252, 14, C.primary, 1.6));
  parts.push(text(L + 36, 264, '已深度思考 12 秒', { size: 11.6, weight: 600, fill: C.ink2 }));
  parts.push(text(R - 46, 264, '展开', { size: 10.2, fill: C.subtle, anchor: 'end' }));
  parts.push(icon('chevronRight', R - 34, 253, 13, C.faint, 2));

  parts.push(text(L, 306, '把超时、重试与退避抽成统一配置，代理层只负责按配置执行；', { size: 13.4, fill: C.body }));
  parts.push(text(L, 330, '流中断后重连用 ', { size: 13.4, fill: C.body }));
  parts.push(text(L + 100, 330, 'Last-Event-ID', { size: 12.4, fill: C.primary2, family: MONO }));
  parts.push(text(L + 190, 330, ' 续传，避免重复计费。', { size: 13.4, fill: C.body }));

  const bullets = [
    '超时按 连接 10s / 首字节 30s / 空闲 45s 分段配置',
    '指数退避 1s → 2s → 4s，最多 3 次；4xx 不重试',
    '单测覆盖断流、超时、半包与客户端主动取消',
  ];
  bullets.forEach((b, i) => {
    const y = 364 + i * 26;
    parts.push(circle(L + 4, y - 4, 2.6, { fill: C.primary, opacity: 0.85 }));
    parts.push(text(L + 16, y, b, { size: 13.2, fill: C.body }));
  });

  const cx = L, cy = 436, cw = W, ch = 238;
  parts.push(rect(cx, cy, cw, ch, { fill: '#171A21', rx: 14 }));
  parts.push(text(cx + 18, cy + 25, 'internal/sse/proxy.go', { size: 10.4, fill: '#8B96A8', family: MONO }));
  parts.push(text(cx + cw - 44, cy + 25, 'Go', { size: 9.4, fill: '#5F6B7C', anchor: 'end', family: MONO }));
  parts.push(icon('copy', cx + cw - 30, cy + 13, 15, '#7F8C98', 1.7));
  parts.push(line(cx, cy + 38, cx + cw, cy + 38, { stroke: '#252A34' }));
  CODE.forEach((segs, i) => {
    parts.push(codeLine(cx + 20, cy + 62 + i * 16.8, segs, 11));
  });

  parts.push(icon('copy', L, 690, 15, C.subtle, 1.7));
  parts.push(text(L + 22, 702, '复制', { size: 10, fill: C.subtle }));
  parts.push(icon('refresh', L + 66, 690, 15, C.subtle, 1.7));
  parts.push(text(L + 88, 702, '重新生成', { size: 10, fill: C.subtle }));
  parts.push(icon('thumbUp', L + 160, 690, 15, C.subtle, 1.6));
  parts.push(icon('thumbDown', L + 190, 690, 15, C.subtle, 1.6));
  parts.push(text(R, 702, '已生成 1,284 字 · 18 tok/s', { size: 9.8, fill: C.faint, anchor: 'end' }));

  const iy = 726;
  parts.push(rect(L, iy, W, 140, { fill: '#FFFFFF', rx: 16, stroke: '#E2E7EF', filter: 'dsgSoft' }));
  parts.push(text(L + 22, iy + 34, '继续追问，或让 DeepTalk 补充测试用例…', { size: 12.8, fill: C.subtle }));
  parts.push(text(L + 22, iy + 58, '正在生成中，可随时停止', { size: 10, fill: C.faint }));

  parts.push(icon('paperclip', L + 22, iy + 92, 17, C.muted, 1.7));
  parts.push(rect(L + 52, iy + 86, 96, 30, { fill: C.primarySoft, rx: 15 }));
  parts.push(icon('spark', L + 62, iy + 94, 13, C.primary, 1.6));
  parts.push(text(L + 80, iy + 105, '深度思考', { size: 11, weight: 600, fill: C.primary }));
  parts.push(rect(L + 158, iy + 86, 96, 30, { fill: '#FFFFFF', rx: 15, stroke: '#E3E7EE' }));
  parts.push(icon('globe', L + 168, iy + 94, 13, C.muted, 1.6));
  parts.push(text(L + 186, iy + 105, '联网搜索', { size: 11, weight: 600, fill: C.ink2 }));

  parts.push(rect(R - 122, iy + 86, 100, 32, { fill: C.primarySoft, rx: 16 }));
  parts.push(rect(R - 104, iy + 95, 12, 12, { fill: C.primary, rx: 2 }));
  parts.push(text(R - 84, iy + 106, '停止生成', { size: 10.8, weight: 600, fill: C.primary }));

  return g(...parts);
}

const inner = g(
  chatSidebar({ active: 'SSE 超时与重连策略' }),
  chatTopbar({ title: 'SSE 超时与重连策略', draft: false }),
  thread(),
  pinLabel(1, 1218, 132),
  pinLabel(2, 1218, 259),
  pinLabel(3, 470, 322),
  pinLabel(4, 1218, 505),
  pinLabel(5, 470, 700),
  pinLabel(6, 1218, 800),
);

const notes = [
  { n: 1, title: '消息列', lines: ['704px 定宽居中；用户右对齐', '浅灰气泡，助手左对齐无气泡。'] },
  { n: 2, title: '深度思考', lines: ['默认收起只显示耗时；展开为', '灰底 reasoning，可随时中断。'] },
  { n: 3, title: '正文排版', lines: ['13.4 / 24，段距 12；行内代码', '等宽主色，列表缩进 16。'] },
  { n: 4, title: '代码块', lines: ['#171A21，头部文件名 + 复制；', '长行横向滚动，不自动换行。'] },
  { n: 5, title: '消息操作', lines: ['复制 / 重新生成 / 赞踩，hover', '才出现，键盘可聚焦。'] },
  { n: 6, title: '生成态', lines: ['输入框切「停止生成」；Esc 中断，', '已输出内容保留，可重试。'] },
];

writeSvg('docs/design/02-chat-streaming.svg', sheetHeader({
  title: '对话进行态 · 流式回答',
  sub: '一次完整回答：思考可折叠、代码可复制、生成可中断',
  code: 'DSG-02 / CHAT-STREAMING',
}) + frame(32, 142, 1440, 900, 'screen02', inner) + specRail({
  items: notes,
  footerLines: ['流式增量按 16ms 合帧渲染', '滚动跟随底部，手动上滚即暂停', '错误：内联提示 + 重试按钮'],
}), 1800, 1080);

