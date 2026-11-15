<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { ActionRunner } from '@/composables/useActions'
import { formatMs } from '@/composables/useFormatters'

const props = defineProps<{ runner: ActionRunner }>()

const answer = ref('')
const input = ref<HTMLInputElement | null>(null)

watch(() => props.runner.pending, async p => {
  answer.value = ''
  if (p) {
    await nextTick()
    input.value?.focus()
  }
})

function submitMid() {
  if (answer.value.trim().toLowerCase() === 'y') props.runner.confirm()
  else props.runner.cancel()
}

function submitHigh() {
  if (answer.value.trim() === props.runner.systemId) props.runner.confirm()
}
</script>

<template>
  <div v-if="runner.pending || runner.result" class="mt-1.5 space-y-0.5 whitespace-pre-wrap break-words">
    <form
      v-if="runner.pending?.risk === 'mid'"
      class="flex flex-wrap items-baseline gap-x-2"
      @submit.prevent="submitMid"
    >
      <span><span class="warn">?</span> {{ runner.pending.label }} <span class="dim">[y/N]</span></span>
      <input
        ref="input"
        v-model="answer"
        class="tin w-[3ch]"
        maxlength="1"
        aria-label="confirm y or n"
        @keydown.esc.prevent="runner.cancel()"
      >
      <button type="button" class="btn" @click="runner.confirm()">y</button>
      <button type="button" class="btn" @click="runner.cancel()">N</button>
    </form>

    <form
      v-else-if="runner.pending?.risk === 'high'"
      class="border-l-2 border-crit pl-2"
      @submit.prevent="submitHigh"
    >
      <div><span class="crit">!</span> {{ runner.pending.label }}</div>
      <div class="flex flex-wrap items-baseline gap-x-2">
        <span class="dim">type <span class="crit">{{ runner.systemId }}</span> to confirm:</span>
        <input
          ref="input"
          v-model="answer"
          class="tin w-[18ch] max-w-full border-b border-line"
          autocomplete="off"
          spellcheck="false"
          :aria-label="`type ${runner.systemId} to confirm`"
          @keydown.esc.prevent="runner.cancel()"
        >
        <button type="submit" class="btn danger" :disabled="answer.trim() !== runner.systemId">run</button>
        <button type="button" class="btn" @click="runner.cancel()">cancel</button>
      </div>
    </form>

    <div v-if="runner.result" role="status">
      <template v-if="runner.result.ok">
        <span class="ok">✓ {{ runner.result.label }} · done {{ formatMs(runner.result.ms) }}</span>
        <span v-if="runner.result.text" class="dim"> {{ runner.result.text }}</span>
      </template>
      <span v-else class="crit">✖ {{ runner.result.label }} · {{ runner.result.text }}</span>
    </div>
  </div>
</template>
