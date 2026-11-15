<script setup lang="ts">
import { computed, reactive } from 'vue'
import type { SystemStatus } from '@/types/metrics'

const props = withDefaults(defineProps<{
  /** Stable key; the collapsed state survives re-renders and route changes. */
  id: string
  title: string
  host?: string
  meta?: string
  tone?: SystemStatus
  collapsible?: boolean
}>(), { collapsible: true })

const collapsed = computed(() => props.collapsible && collapsedById[props.id] === true)
const toggle = () => { collapsedById[props.id] = !collapsed.value }
</script>

<script lang="ts">
// Shared by every block instance, so collapse state outlives a single component.
const collapsedById = reactive<Record<string, boolean>>({})
</script>

<template>
  <section
    class="blk"
    :class="{ collapsed, 'is-warning': tone === 'warning', 'is-critical': tone === 'critical' }"
  >
    <header class="cmd">
      <button
        v-if="collapsible"
        type="button"
        class="min-w-0 truncate text-left"
        :aria-expanded="!collapsed"
        @click="toggle"
      >
        <span class="p">❯</span> {{ title }}<span v-if="host" class="dim"> @{{ host }}</span>
        <span class="dim"> {{ collapsed ? '▸' : '▾' }}</span>
      </button>
      <span v-else class="min-w-0 truncate">
        <span class="p">❯</span> {{ title }}<span v-if="host" class="dim"> @{{ host }}</span>
      </span>
      <span class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <slot name="actions" />
        <span v-if="meta || $slots.meta" class="dim"><slot name="meta">{{ meta }}</slot></span>
      </span>
    </header>
    <div v-if="!collapsed" class="out">
      <slot />
    </div>
  </section>
</template>
