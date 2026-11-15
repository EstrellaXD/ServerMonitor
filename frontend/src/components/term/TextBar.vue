<script setup lang="ts">
import { computed } from 'vue'
import { formatBar, usageLevel } from '@/composables/useFormatters'

const props = withDefaults(defineProps<{ percent: number; width?: number; warn?: number; crit?: number }>(), {
  width: 20,
  warn: 75,
  crit: 90,
})

const text = computed(() => formatBar(props.percent, props.width))
const level = computed(() => usageLevel(props.percent, props.warn, props.crit))
</script>

<template>
  <span :class="level" aria-hidden="true">{{ text }}</span>
</template>
