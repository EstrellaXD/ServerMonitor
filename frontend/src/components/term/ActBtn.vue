<script setup lang="ts">
import { computed } from 'vue'
import type { ActionParams, ActionRunner } from '@/composables/useActions'

const props = defineProps<{
  runner: ActionRunner
  action: string
  target: string
  params?: ActionParams
  /** Text inside the brackets; defaults to the action name. */
  text?: string
  /** Log/prompt label; defaults to "<action> <target>". */
  label?: string
  active?: boolean
}>()

const risk = computed(() => props.runner.risk(props.action, props.params))
const busy = computed(() => props.runner.isBusy(props.action, props.target))
</script>

<template>
  <button
    v-if="runner.can(action, params)"
    type="button"
    class="btn"
    :class="{ danger: risk === 'high', on: active }"
    :disabled="busy"
    @click.stop="runner.request(action, target, { params, label })"
  >{{ busy ? '…' : (text ?? action) }}</button>
</template>
