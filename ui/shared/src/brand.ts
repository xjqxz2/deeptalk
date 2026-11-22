/**
 * 品牌标记：统一使用小橘图标（透明底橘色小橘子），C 端与 B 端共用同一份资源。
 * 说明：图标是位图，通过 Vite 资源导入得到带哈希的 URL，构建与开发期都能正确解析。
 */
import { h, type FunctionalComponent } from 'vue'
import brandMark from './assets/brand-mark.png'

/** 品牌标记组件，size 控制边长；圆角由图标自身与容器共同保证。 */
export const BrandMark: FunctionalComponent<{ size?: number; radius?: number }> = (props) => {
  const size = props.size ?? 30
  return h('span', {
    class: 'dt-brand-mark',
    style: {
      width: `${size}px`,
      height: `${size}px`,
    },
  }, [
    h('img', {
      src: brandMark,
      alt: '小橘',
      width: size,
      height: size,
      draggable: false,
      decoding: 'async',
    }),
  ])
}

BrandMark.props = { size: { type: Number, default: 30 }, radius: { type: Number, default: 0 } }
