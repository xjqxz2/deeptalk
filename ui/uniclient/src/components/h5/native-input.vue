<script>
/** 原生 input，避免 uni-input 的元素映射与内置样式。 */
import { h, ref } from 'vue'

export default {
  name: 'NativeInput',
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '' },
    ariaLabel: { type: String, default: '' },
    type: { type: String, default: 'text' },
    autocomplete: { type: String, default: 'off' },
    name: { type: String, default: '' },
    fieldClass: { type: String, default: 'native-input' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, expose }) {
    const el = ref(null)
    expose({ focus: () => el.value && el.value.focus() })
    return () =>
      h('input', {
        ref: el,
        class: props.fieldClass,
        type: props.type,
        autocomplete: props.autocomplete,
        name: props.name || undefined,
        value: props.modelValue,
        placeholder: props.placeholder,
        'aria-label': props.ariaLabel || props.placeholder,
        onInput: (event) => emit('update:modelValue', event.target.value),
      })
  },
}
</script>
