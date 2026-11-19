import { C, text, rect, circle, line, icon, brandMark } from '../requirements/req-lib.mjs';
import { g } from './design-lib.mjs';

export function authLeft() {
  const parts = [];
  parts.push('<defs><linearGradient id="authSide" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#F3F6FF"/><stop offset="0.55" stop-color="#EEF2FF"/><stop offset="1" stop-color="#E7EDFF"/></linearGradient></defs>');
  parts.push(rect(0, 0, 620, 900, { fill: 'url(#authSide)' }));
  parts.push(circle(566, -20, 214, { fill: '#DFE8FF', opacity: 0.55 }));
  parts.push(circle(74, 872, 186, { fill: '#E4EBFF', opacity: 0.5 }));
  parts.push(brandMark(48, 44, 32));
  parts.push(text(90, 67, 'DeepTalk', { size: 16.5, weight: 730, spacing: -0.3 }));
  parts.push(text(90, 82, 'PERSONAL AI · MVP', { size: 8, fill: C.subtle, weight: 620, spacing: 1.3 }));

  parts.push(text(48, 292, '让每一次对话，', { size: 34, weight: 730, spacing: -0.6 }));
  parts.push(text(48, 340, '都更接近答案。', { size: 34, weight: 730, spacing: -0.6 }));
  parts.push(text(48, 396, 'DeepTalk 是你的个人 AI 工作台，', { size: 13, fill: C.muted }));
  parts.push(text(48, 418, '写代码、读文档、做分析，也能一起头脑风暴。', { size: 13, fill: C.muted }));

  const feats = [
    ['spark', '深度思考与代码块原生支持'],
    ['shield', '会话与用量按账号隔离'],
    ['server', '管理员统一配置上游模型'],
  ];
  feats.forEach((f, i) => {
    const y = 480 + i * 38;
    parts.push(rect(48, y - 14, 26, 26, { fill: '#FFFFFF', rx: 8, opacity: 0.85 }));
    parts.push(icon(f[0], 53, y - 9, 16, C.primary2, 1.8));
    parts.push(text(84, y + 4, f[1], { size: 11.6, fill: C.ink2 }));
  });

  parts.push(line(48, 800, 572, 800, { stroke: '#D9E2F7' }));
  parts.push(text(48, 826, '© 2026 DeepTalk', { size: 9.4, fill: C.subtle }));
  parts.push(text(572, 826, '隐私 · 条款', { size: 9.4, fill: C.subtle, anchor: 'end' }));
  return g(...parts);
}

export function authRight(children) {
  return g(
    rect(620, 0, 820, 900, { fill: '#FFFFFF' }),
    line(620, 0, 620, 900, { stroke: '#E9EDF4' }),
    children,
  );
}

export function topLink(left, right) {
  return g(
    text(1360, 56, left, { size: 10.6, fill: C.muted, anchor: 'end' }),
    text(1416, 56, right, { size: 10.6, weight: 650, fill: C.primary, anchor: 'end' }),
  );
}
