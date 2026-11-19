/**
 * 品牌标记。设计稿中是一个圆角渐变方块 + 白色 D 形线条；
 * 这里用函数式组件复用，避免两个应用各画一份。
 */
import { h, type FunctionalComponent } from 'vue'

/** 静态 SVG，内容完全可控，因此可以直接写入 innerHTML。 */
const BRAND_SVG = `
<svg viewBox="0 0 30 30" width="100%" height="100%" role="img" aria-label="DeepTalk">
  <defs>
    <linearGradient id="dt-brand-grad" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#6D86FF"/>
      <stop offset="1" stop-color="#3F5BF6"/>
    </linearGradient>
  </defs>
  <rect width="30" height="30" rx="8.4" fill="url(#dt-brand-grad)"/>
  <g fill="none" stroke="#fff" stroke-width="2.1" stroke-linecap="round" stroke-linejoin="round">
    <path d="M10.6 9.2h3.1a5.2 5.2 0 0 1 0 10.4h-3.1V9.2Z"/>
    <path d="M10.6 9.2v10.4"/>
    <circle cx="18.9" cy="10.2" r="1.05" fill="#fff" stroke="none"/>
  </g>
</svg>`

/** 品牌标记组件，size 控制边长。 */
export const BrandMark: FunctionalComponent<{ size?: number; radius?: number }> = (props) => {
  const size = props.size ?? 30
  const radius = props.radius ?? Math.round(size * 0.28)
  return h('span', {
    class: 'dt-brand-mark',
    style: {
      width: `${size}px`,
      height: `${size}px`,
      borderRadius: `${radius}px`,
    },
    innerHTML: BRAND_SVG,
  })
}

BrandMark.props = { size: { type: Number, default: 30 }, radius: { type: Number, default: 0 } }
