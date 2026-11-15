<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import type { DockerLogLine } from '@/types/api'
import { fetchDockerLogs } from '@/composables/useActions'
import { formatClock, formatMs } from '@/composables/useFormatters'
import { useFilter } from '@/composables/useFilter'
import TermBlock from '@/components/term/TermBlock.vue'

const props = defineProps<{ systemId: string; container: string }>()
defineEmits<{ close: [] }>()

const TAIL = 200

const { matches } = useFilter()
const lines = ref<DockerLogLine[]>([])
const error = ref<string | null>(null)
const loading = ref(false)
const fetchedAt = ref<Date | null>(null)
const took = ref(0)
const pane = ref<HTMLElement | null>(null)

const shown = computed(() => lines.value.filter(l => matches(l.text)))
const meta = computed(() => {
  if (loading.value) return 'loading…'
  if (!fetchedAt.value) return ''
  const n = shown.value.length === lines.value.length ? `${lines.value.length}` : `${shown.value.length}/${lines.value.length}`
  return `${n} lines · ${formatMs(took.value)} · ${formatClock(fetchedAt.value)}`
})

async function load() {
  loading.value = true
  const t0 = performance.now()
  try {
    const res = await fetchDockerLogs(props.systemId, props.container, TAIL)
    lines.value = res.lines
    error.value = null
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    took.value = performance.now() - t0
    fetchedAt.value = new Date()
    loading.value = false
  }
  await nextTick()
  if (pane.value) pane.value.scrollTop = pane.value.scrollHeight
}

onMounted(load)
</script>

<template>
  <TermBlock :id="`${systemId}:logs:${container}`" :title="`docker logs --tail ${TAIL} ${container}`" :meta="meta">
    <template #actions>
      <button type="button" class="btn" :disabled="loading" @click="load">refresh</button>
      <button type="button" class="btn" @click="$emit('close')">close</button>
    </template>
    <div v-if="error" class="crit">✖ {{ error }}</div>
    <div v-else ref="pane" class="max-h-[28rem] overflow-auto">
      <div v-if="!shown.length && !loading" class="dim">(no output)</div>
      <div v-for="(l, i) in shown" :key="i" class="whitespace-pre" :class="{ crit: l.stream === 'stderr' }">{{ l.text }}</div>
    </div>
  </TermBlock>
</template>
