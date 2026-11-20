<script>
/**
 * 原生 button。
 * uni-app 会把模板里的 <button> 编译成 uni-button 自定义元素，并套上自己的
 * 边框与底色，和 PWA 的按钮样式冲突。用渲染函数创建就绕开了这层映射。
 */
import { h } from 'vue'

export default {
  name: 'NativeButton',
  props: {
    disabled: { type: Boolean, default: false },
    type: { type: String, default: 'button' },
    ariaLabel: { type: String, default: '' },
    title: { type: String, default: '' },
  },
  emits: ['click'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          type: props.type,
          disabled: props.disabled,
          'aria-label': props.ariaLabel || undefined,
          title: props.title || undefined,
          onClick: (event) => emit('click', event),
        },
        slots.default ? slots.default() : undefined,
      )
  },
}
</script>
