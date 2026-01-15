<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useTheme } from '@/composables/useTheme'
import { useWebSocket } from '@/composables/useWebSocket'
import ThemeToggle from '@/components/common/ThemeToggle.vue'
import ConnectionStatus from '@/components/common/ConnectionStatus.vue'

useTheme()
const { connected, reconnecting } = useWebSocket()

const isOnline = ref(navigator.onLine)
const handleOnline = () => { isOnline.value = true }
const handleOffline = () => { isOnline.value = false }

onMounted(() => {
  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
})
onUnmounted(() => {
  window.removeEventListener('online', handleOnline)
  window.removeEventListener('offline', handleOffline)
})
</script>

<template>
  <div class="min-h-screen bg-surface-100 dark:bg-surface-950 transition-colors duration-300">
    <!-- Offline banner -->
    <div
      v-if="!isOnline"
      class="fixed top-0 inset-x-0 z-50 bg-amber-500 text-black text-center text-sm py-1 font-medium"
    >
      Offline — showing cached app shell
    </div>

    <!-- Ambient background gradient -->
    <div class="fixed inset-0 -z-10 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 bg-primary-500/10 dark:bg-primary-500/5 rounded-full blur-3xl"></div>
      <div class="absolute top-1/2 -left-40 w-80 h-80 bg-primary-600/10 dark:bg-primary-600/5 rounded-full blur-3xl"></div>
    </div>

    <!-- Navbar -->
    <header class="navbar">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <!-- Logo -->
          <router-link to="/" class="flex items-center gap-2 sm:gap-3 group">
            <div class="w-9 h-9 rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center shadow-lg shadow-primary-500/25 group-hover:shadow-primary-500/40 transition-shadow">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
                <line x1="8" y1="21" x2="16" y2="21"></line>
                <line x1="12" y1="17" x2="12" y2="21"></line>
              </svg>
            </div>
            <span class="text-lg sm:text-xl font-heading font-semibold text-slate-900 dark:text-white">
              <span class="hidden xs:inline">Server</span><span class="text-primary-500">Monitor</span>
            </span>
          </router-link>

          <!-- Right side controls -->
          <div class="flex items-center gap-2 sm:gap-4">
            <ConnectionStatus :connected="connected" :reconnecting="reconnecting" />
            <div class="w-px h-6 bg-slate-200 dark:bg-white/10 hidden sm:block"></div>
            <ThemeToggle />
          </div>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease-out;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
