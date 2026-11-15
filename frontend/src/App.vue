<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { useWebSocket } from '@/composables/useWebSocket'
import { useFilter } from '@/composables/useFilter'
import { useMetricsStore } from '@/stores/metrics'
import { formatClock } from '@/composables/useFormatters'
import SystemNav from '@/components/SystemNav.vue'

const { isDark, toggleTheme } = useTheme()
const { connected, reconnecting } = useWebSocket()
const { query, inputEl, focus } = useFilter()
const store = useMetricsStore()
const router = useRouter()

const isOnline = ref(navigator.onLine)
const filterFocused = ref(false)

const wsState = computed(() => {
  if (connected.value) return { glyph: '●', cls: 'ok', text: 'ws connected' }
  if (reconnecting.value) return { glyph: '▲', cls: 'warn', text: 'ws reconnecting · http poll' }
  return { glyph: '○', cls: 'off', text: 'ws closed' }
})

const isTyping = (t: EventTarget | null) =>
  t instanceof HTMLElement && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName))

function onKey(e: KeyboardEvent) {
  if (e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target)) return
  if (e.key >= '1' && e.key <= '9') {
    const s = store.systemsList[Number(e.key) - 1]
    if (s) router.push(`/system/${s.id}`)
  } else if (e.key === 'Escape') {
    router.push('/')
  } else if (e.key === '/') {
    e.preventDefault()
    focus()
  } else if (e.key === 't') {
    toggleTheme()
  }
}

function onFilterEsc() {
  if (query.value) query.value = ''
  else inputEl.value?.blur()
}

const setOnline = () => { isOnline.value = navigator.onLine }

onMounted(() => {
  window.addEventListener('keydown', onKey)
  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOnline)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOnline)
})
</script>

<template>
  <div class="min-h-screen sm:grid sm:grid-cols-[176px_minmax(0,1fr)]">
    <SystemNav />

    <div class="flex min-h-screen min-w-0 flex-col">
      <main class="flex flex-1 flex-col gap-2.5 px-3 py-2.5 sm:px-4">
        <router-view />
      </main>

      <footer class="sticky bottom-0 space-y-1.5 border-t border-line bg-bg px-3 py-2 sm:px-4">
        <label class="blk flex cursor-text items-baseline overflow-hidden px-2.5 py-1.5">
          <span class="ok mr-[1ch]">❯</span><span class="dim mr-[1ch]">filter:</span>
          <input
            ref="inputEl"
            v-model="query"
            class="tin max-w-full"
            :style="{ width: `${query.length + 1}ch` }"
            spellcheck="false"
            autocomplete="off"
            aria-label="filter rows"
            @focus="filterFocused = true"
            @blur="filterFocused = false"
            @keydown.esc.prevent="onFilterEsc"
          ><span v-if="!filterFocused" class="caret" aria-hidden="true">&nbsp;</span>
          <span v-if="!query && !filterFocused" class="dim ml-[1ch] truncate">press / to filter</span>
        </label>
        <div class="dim flex flex-wrap justify-between gap-x-4 text-xs">
          <span>
            <span :class="wsState.cls">{{ wsState.glyph }}</span> {{ wsState.text }}
            · updated {{ formatClock(store.lastUpdate) }}
            <span v-if="!isOnline" class="crit"> · browser offline</span>
          </span>
          <span class="hidden sm:inline">[1-9] system · [esc] overview · [/] filter · [t] {{ isDark ? 'light' : 'dark' }}</span>
        </div>
      </footer>
    </div>
  </div>
</template>
