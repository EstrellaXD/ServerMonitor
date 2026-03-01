<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

export interface MenuItem {
  label: string
  action: string
  variant?: 'default' | 'danger'
}

defineProps<{
  items: MenuItem[]
  loading?: boolean
}>()

const emit = defineEmits<{
  select: [action: string]
}>()

const open = ref(false)
const menuRef = ref<HTMLElement | null>(null)

const toggle = () => {
  open.value = !open.value
}

const select = (action: string) => {
  open.value = false
  emit('select', action)
}

const onClickOutside = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', onClickOutside)
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div ref="menuRef" class="relative">
    <!-- Trigger: loading spinner or kebab dots -->
    <button
      v-if="loading"
      class="w-8 h-8 rounded-lg flex items-center justify-center"
      disabled
      aria-label="Action in progress"
    >
      <svg class="w-4 h-4 text-primary-500 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </button>
    <button
      v-else
      @click.stop="toggle"
      class="w-8 h-8 rounded-lg flex items-center justify-center
             hover:bg-slate-100 dark:hover:bg-surface-700
             transition-colors duration-150 cursor-pointer"
      :aria-expanded="open"
      aria-label="Actions menu"
      aria-haspopup="true"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-500 dark:text-slate-400" viewBox="0 0 24 24" fill="currentColor">
        <circle cx="12" cy="5" r="2" />
        <circle cx="12" cy="12" r="2" />
        <circle cx="12" cy="19" r="2" />
      </svg>
    </button>

    <!-- Dropdown -->
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95 -translate-y-1"
      enter-to-class="opacity-100 scale-100 translate-y-0"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100 translate-y-0"
      leave-to-class="opacity-0 scale-95 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute right-0 top-full mt-1 z-50
               w-44 py-1
               bg-white/80 dark:bg-surface-800/80
               backdrop-blur-md
               border border-slate-200/60 dark:border-white/[0.08]
               rounded-xl shadow-lg dark:shadow-black/20
               origin-top-right"
        role="menu"
      >
        <button
          v-for="item in items"
          :key="item.action"
          @click.stop="select(item.action)"
          class="w-full px-3 py-2 text-left text-sm font-medium
                 flex items-center gap-2.5
                 transition-colors duration-150 cursor-pointer"
          :class="[
            item.variant === 'danger'
              ? 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10'
              : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/[0.06]'
          ]"
          role="menuitem"
        >
          <!-- Start / Resume -->
          <svg v-if="item.action === 'start' || item.action === 'resume'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          <!-- Stop -->
          <svg v-else-if="item.action === 'stop'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
          </svg>
          <!-- Restart -->
          <svg v-else-if="item.action === 'restart'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10" />
            <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
          </svg>
          <!-- Pause -->
          <svg v-else-if="item.action === 'pause'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="6" y="4" width="4" height="16" />
            <rect x="14" y="4" width="4" height="16" />
          </svg>
          <!-- Delete -->
          <svg v-else-if="item.action === 'delete'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="3 6 5 6 21 6" />
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
          </svg>
          {{ item.label }}
        </button>
      </div>
    </Transition>
  </div>
</template>
