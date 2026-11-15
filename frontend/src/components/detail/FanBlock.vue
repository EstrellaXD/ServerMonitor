<script setup lang="ts">
import { computed } from 'vue'
import type { FanMetrics, SystemMetrics } from '@/types/metrics'
import { useActions } from '@/composables/useActions'
import { useFilter } from '@/composables/useFilter'
import TermBlock from '@/components/term/TermBlock.vue'
import TextBar from '@/components/term/TextBar.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'

const props = defineProps<{ system: SystemMetrics; fans: FanMetrics[] }>()

const STEPS = [0, 25, 50, 75, 100]

const { matches } = useFilter()
const runner = useActions(() => props.system)
const shown = computed(() => props.fans.filter(f => matches(f.label, f.mode)))
const isStep = (f: FanMetrics, step: number) =>
  f.mode === 'manual' && f.pwm_percent != null && Math.round(f.pwm_percent) === step
</script>

<template>
  <TermBlock :id="`${system.id}:fans`" title="fan" :meta="`${fans.length} fan${fans.length === 1 ? '' : 's'}`">
    <table class="tbl">
      <thead>
        <tr><th>FAN</th><th class="r">RPM</th><th>PWM</th><th class="r">%</th><th>MODE</th><th v-if="runner.can('fan_set')" /></tr>
      </thead>
      <tbody>
        <tr v-for="f in shown" :key="f.label">
          <td>{{ f.label }}</td>
          <td class="r">{{ f.rpm }}</td>
          <td><TextBar v-if="f.pwm_percent != null" :percent="f.pwm_percent" :width="10" :warn="101" :crit="101" /><span v-else class="dim">–</span></td>
          <td class="r">{{ f.pwm_percent != null ? f.pwm_percent.toFixed(0) : '–' }}</td>
          <td :class="f.mode === 'manual' ? 'warn' : 'dim'">{{ f.mode ?? '–' }}</td>
          <td v-if="runner.can('fan_set')"><template v-if="f.mode">
            <ActBtn :runner="runner" action="fan_auto" :target="f.label" text="auto" :active="f.mode === 'auto'" :label="`fan ${f.label} auto`" />
            <ActBtn
              v-for="step in STEPS"
              :key="step"
              :runner="runner"
              action="fan_set"
              :target="f.label"
              :params="{ percent: step }"
              :text="String(step)"
              :active="isStep(f, step)"
              :label="`fan ${f.label} ${step}%`"
            />
          </template></td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="runner" />
  </TermBlock>
</template>
