<script setup lang="ts">
defineProps<{
  connected: boolean
  reconnecting: boolean
}>()
</script>

<template>
  <div class="flex items-center gap-2.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-white/5">
    <span
      class="relative flex h-2.5 w-2.5"
    >
      <!-- Ping animation for connected -->
      <span
        v-if="connected"
        class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"
      ></span>
      <!-- Static dot -->
      <span
        class="relative inline-flex rounded-full h-2.5 w-2.5"
        :class="{
          'bg-emerald-500': connected,
          'bg-amber-500': reconnecting,
          'bg-red-500': !connected && !reconnecting
        }"
      ></span>
    </span>
    <span class="text-sm font-medium text-slate-600 dark:text-slate-300">
      {{ connected ? 'Live' : reconnecting ? 'Reconnecting...' : 'Offline' }}
    </span>
  </div>
</template>
