<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps<{
  show: boolean
  title: string
  message: string
  confirmLabel?: string
  variant?: 'warning' | 'danger'
  showDeleteFiles?: boolean
}>()

const emit = defineEmits<{
  confirm: [deleteFiles?: boolean]
  cancel: []
}>()

const deleteFiles = ref(false)
const backdropRef = ref<HTMLElement | null>(null)

const confirm = () => {
  emit('confirm', props.showDeleteFiles ? deleteFiles.value : undefined)
  deleteFiles.value = false
}

const cancel = () => {
  emit('cancel')
  deleteFiles.value = false
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.show) {
    cancel()
  }
}

watch(() => props.show, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
})

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="show"
        ref="backdropRef"
        class="fixed inset-0 z-[100] flex items-center justify-center p-4"
        @click.self="cancel"
      >
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" />

        <!-- Modal -->
        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 scale-95 translate-y-2"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 scale-100 translate-y-0"
          leave-to-class="opacity-0 scale-95 translate-y-2"
        >
          <div
            v-if="show"
            class="relative w-full max-w-sm
                   bg-white/90 dark:bg-surface-800/90
                   backdrop-blur-md
                   border border-slate-200/60 dark:border-white/[0.08]
                   rounded-2xl shadow-xl dark:shadow-black/30
                   p-6"
            role="dialog"
            aria-modal="true"
            :aria-label="title"
          >
            <!-- Icon -->
            <div class="flex justify-center mb-4">
              <div
                class="w-12 h-12 rounded-xl flex items-center justify-center"
                :class="variant === 'danger' ? 'bg-red-500/10' : 'bg-amber-500/10'"
              >
                <!-- Warning icon -->
                <svg
                  v-if="variant === 'warning'"
                  xmlns="http://www.w3.org/2000/svg" class="w-6 h-6 text-amber-500"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                >
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <line x1="12" y1="9" x2="12" y2="13" />
                  <line x1="12" y1="17" x2="12.01" y2="17" />
                </svg>
                <!-- Danger icon -->
                <svg
                  v-else
                  xmlns="http://www.w3.org/2000/svg" class="w-6 h-6 text-red-500"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                >
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
              </div>
            </div>

            <!-- Content -->
            <div class="text-center mb-6">
              <h3 class="text-lg font-heading font-semibold text-slate-900 dark:text-white mb-2">
                {{ title }}
              </h3>
              <p class="text-sm text-slate-500 dark:text-slate-400">
                {{ message }}
              </p>
            </div>

            <!-- Delete files checkbox -->
            <div v-if="showDeleteFiles" class="mb-6">
              <label class="flex items-center gap-3 cursor-pointer group">
                <input
                  v-model="deleteFiles"
                  type="checkbox"
                  class="w-4 h-4 rounded border-slate-300 dark:border-slate-600
                         text-red-500 focus:ring-red-500/30
                         bg-white dark:bg-surface-900 cursor-pointer"
                />
                <span class="text-sm text-slate-700 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white transition-colors">
                  Also delete downloaded files
                </span>
              </label>
            </div>

            <!-- Actions -->
            <div class="flex gap-3">
              <button
                @click="cancel"
                class="flex-1 px-4 py-2.5 text-sm font-medium rounded-xl
                       text-slate-700 dark:text-slate-300
                       hover:bg-slate-100 dark:hover:bg-white/[0.06]
                       transition-colors duration-150 cursor-pointer"
              >
                Cancel
              </button>
              <button
                @click="confirm"
                class="flex-1 px-4 py-2.5 text-sm font-medium rounded-xl text-white
                       transition-all duration-150 cursor-pointer
                       active:scale-[0.98]"
                :class="[
                  variant === 'danger'
                    ? 'bg-red-500 hover:bg-red-600 shadow-md hover:shadow-glow-red'
                    : 'bg-amber-500 hover:bg-amber-600 shadow-md hover:shadow-glow-amber'
                ]"
              >
                {{ confirmLabel || 'Confirm' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
