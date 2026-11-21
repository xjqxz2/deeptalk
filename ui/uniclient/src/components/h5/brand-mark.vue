<template>
  <img class="brand-mark" :src="logoSrc" :width="size" :height="size" :style="{ borderRadius: radius + 'px' }" alt="小橘树洞" />
</template>

<script setup>
/** 品牌标记：与 PWA 的 BrandMark 使用同一张鲨鱼图。 */
const props = defineProps({
  size: { type: Number, default: 28 },
  radius: { type: Number, default: 8 },
})
const logoSrc = '/static/logo.png'
</script>
