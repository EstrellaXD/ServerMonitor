<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  max: number
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
  variant?: 'default' | 'gradient'
  colorMode?: 'usage' | 'progress'
}>(), {
  showLabel: false,
  size: 'md',
  variant: 'default',
  colorMode: 'usage',
})

const percent = computed(() => {
  if (props.max === 0) return 0
  return Math.min(100, (props.value / props.max) * 100)
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'h-1.5'
    case 'lg': return 'h-3'
    default: return 'h-2'
  }
})

const colorClass = computed(() => {
  // Progress mode: always use primary color (high values are good)
  if (props.colorMode === 'progress') {
    if (props.variant === 'gradient') {
      return 'bg-gradient-to-r from-primary-500 to-primary-400'
    }
    return 'bg-primary-500'
  }

  // Usage mode: red at 90%+, amber at 75%+ (high values are warnings)
  if (props.variant === 'gradient') {
    if (percent.value >= 90) return 'bg-gradient-to-r from-red-500 to-red-400'
    if (percent.value >= 75) return 'bg-gradient-to-r from-amber-500 to-amber-400'
    return 'bg-gradient-to-r from-primary-500 to-primary-400'
  }

  if (percent.value >= 90) return 'bg-red-500'
  if (percent.value >= 75) return 'bg-amber-500'
  return 'bg-primary-500'
})
</script>

<template>
  <div class="w-full">
    <div
      class="progress-track"
      :class="sizeClass"
    >
      <div
        class="progress-fill"
        :class="colorClass"
        :style="{ width: `${percent}%` }"
      ></div>
    </div>
    <div v-if="showLabel" class="flex justify-between items-center mt-1.5">
      <span class="text-xs text-slate-500 dark:text-slate-400">{{ value.toFixed(1) }}%</span>
      <span class="text-xs text-slate-400 dark:text-slate-500">{{ max }}%</span>
    </div>
  </div>
</template>
