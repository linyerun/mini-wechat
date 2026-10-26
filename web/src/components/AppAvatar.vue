<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ name?: string; src?: string; size?: number; square?: boolean }>(),
  { name: '?', src: '', size: 40, square: false },
)

const palette = [
  '#07c160',
  '#10aeff',
  '#ff9f0a',
  '#ff5f57',
  '#7d5fff',
  '#00b8a9',
  '#f76b8a',
  '#5b8def',
]

const paletteIndex = computed(() => {
  let hash = 0
  for (const ch of props.name) hash = (hash * 31 + (ch.codePointAt(0) ?? 0)) >>> 0
  return hash % palette.length
})

const background = computed(
  () =>
    `linear-gradient(135deg, ${palette[paletteIndex.value]}, ${
      palette[(paletteIndex.value + 3) % palette.length]
    })`,
)
const initial = computed(() => props.name.trim().slice(0, 1).toUpperCase() || '?')
const fontSize = computed(() => Math.max(12, Math.round(props.size * 0.42)))
const radius = computed(() => (props.square ? `${Math.round(props.size * 0.28)}px` : '50%'))
</script>

<template>
  <div
    class="avatar"
    :style="{
      width: `${size}px`,
      height: `${size}px`,
      background: src ? '#e5e5e5' : background,
      borderRadius: radius,
    }"
  >
    <img v-if="src" :src="src" :alt="name" />
    <span v-else :style="{ fontSize: `${fontSize}px` }">{{ initial }}</span>
  </div>
</template>
