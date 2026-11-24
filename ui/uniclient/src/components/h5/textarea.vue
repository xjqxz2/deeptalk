<script>
/**
 * 原生 textarea。
 * 用渲染函数创建，绕过 uni-app 对 <textarea> 的组件映射——
 * 映射后的 uni-textarea 自带 150px 高度与占位层，和 PWA 的排版对不上。
 */
import { h, ref } from 'vue'

export default {
  name: 'NativeTextarea',
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue', 'submit'],
  setup(props, { emit, expose }) {
    const el = ref(null)

    expose({
      clear() {
        if (el.value) {
          el.value.value = ''
          el.value.style.height = ''
        }
      },
      focus() {
        if (el.value) el.value.focus()
      },
    })

    /** 按内容高度自适应：先归零再读 scrollHeight，避免只增不减。 */
    const autoGrow = (target) => {
      if (!target) return
      target.style.height = 'auto'
      target.style.height = `${Math.min(target.scrollHeight, 192)}px`
    }

    const onInput = (event) => {
      autoGrow(event.target)
      emit('update:modelValue', event.target.value)
    }
    const onKeydown = (event) => {
      if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
      event.preventDefault()
      emit('submit', event.target.value)
    }

    return () =>
      h('textarea', {
        ref: el,
        class: 'composer-input',
        rows: 1,
        value: props.modelValue,
        placeholder: props.placeholder,
        disabled: props.disabled,
        'aria-label': '消息输入框',
        onInput,
        onKeydown,
      })
  },
}
</script>