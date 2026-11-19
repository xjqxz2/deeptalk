import { C, text, rect, circle, line, icon } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, writeSvg, g, field, btn } from './design-lib.mjs';
import { authLeft, authRight, topLink } from './auth-lib.mjs';

const FX = 840, FW = 380, FR = FX + FW;

function form() {
  const parts = [];
  parts.push(topLink('还没有账号？', '立即注册'));
  parts.push(text(FX, 300, 'ACCOUNT ACCESS', { size: 8.8, weight: 620, fill: C.subtle, spacing: 1.5 }));
  parts.push(text(FX, 344, '欢迎回来', { size: 30, weight: 730, spacing: -0.5 }));
  parts.push(text(FX, 372, '登录后继续你的 DeepTalk 工作台。', { size: 12.2, fill: C.muted }));

  parts.push(field(FX, 434, FW, '邮箱', { iconName: 'mail', value: 'lmo@deeptalk.app', focus: true }));
  parts.push(text(FX, 512, '密码', { size: 11.4, weight: 600, fill: C.ink2 }));
  parts.push(text(FR, 512, '忘记密码？', { size: 10.2, fill: C.primary, anchor: 'end' }));
  parts.push(field(FX, 520, FW, '', {
    iconName: 'lock',
    value: '••••••••',
    trailing: icon('eye', FR - 34, 535, 16, C.subtle, 1.7),
  }));

  parts.push(btn(FX, 594, FW, 46, '登录', { primary: true, radius: 11, size: 13.6 }));
  parts.push(circle(FX + 8, 664, 3, { fill: C.green }));
  parts.push(text(FX + 18, 668, '登录后进入你的对话空间', { size: 9.8, fill: C.subtle }));

  parts.push(text(FX, 706, '登录即表示同意', { size: 9.4, fill: C.faint }));
  parts.push(text(FX + 66, 706, '《服务条款》', { size: 9.4, fill: C.muted }));
  parts.push(text(FX + 129, 706, '与', { size: 9.4, fill: C.faint }));
  parts.push(text(FX + 142, 706, '《隐私政策》', { size: 9.4, fill: C.muted }));
  parts.push(text(FX, 744, '无需验证邮件 · 忘记密码由管理员重置', { size: 9.6, fill: C.faint }));

  parts.push(line(FX, 766, FR, 766, { stroke: '#F0F2F6' }));
  parts.push(icon('lock', FX, 786, 13, C.faint, 1.8));
  parts.push(text(FX + 18, 796, '全程 HTTPS · 会话 HttpOnly，默认 7 天', { size: 9.2, fill: C.faint }));

  parts.push(pinLabel(1, 608, 300));
  parts.push(pinLabel(2, 996, 334));
  parts.push(pinLabel(3, 1234, 455));
  parts.push(pinLabel(4, 1234, 617));
  parts.push(pinLabel(5, 826, 744));
  return g(...parts);
}

const inner = g(authLeft(), authRight(form()));

const notes = [
  { n: 1, title: '品牌区', lines: ['左侧 620 固定；<1024 隐藏，', '表单居中并保留品牌小标。'] },
  { n: 2, title: '表单头部', lines: ['标题 30，副文案一行；表单宽', '380，字段 46 高、间距 12。'] },
  { n: 3, title: '邮箱与密码', lines: ['聚焦 2px 主色环；错误统一提示', '「邮箱或密码不正确」，不泄露账号。'] },
  { n: 4, title: '主按钮', lines: ['46 高全宽；提交中 loading 并禁用，', '连续 5 次失败锁 10 分钟。'] },
  { n: 5, title: '无邮件依赖', lines: ['不做验证邮件与邮件找回；忘记', '密码提示联系管理员或 CLI 重置。'] },
];

writeSvg('docs/design/05-auth-login.svg', sheetHeader({
  title: '登录 · 邮箱密码',
  sub: '一屏完成身份确认：品牌定调在左，表单在右，错误不泄露账号',
  code: 'DSG-05 / AUTH-LOGIN',
  meta: '1440 × 900 · Web',
}) + frame(32, 142, 1440, 900, 'screen05', inner) + specRail({
  items: notes,
  footerLines: ['密码可见性切换保留焦点', '错误 aria-live="polite"', '对比度 ≥ 4.5:1，点击区 ≥ 44px'],
}), 1800, 1080);
