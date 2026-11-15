import { ref } from 'vue'

// One filter prompt for the whole app; every view filters its rows with it.
const query = ref('')
const inputEl = ref<HTMLInputElement | null>(null)

export function useFilter() {
  const matches = (...fields: Array<string | number | null | undefined>): boolean => {
    const q = query.value.trim().toLowerCase()
    if (!q) return true
    return fields.some(f => f != null && String(f).toLowerCase().includes(q))
  }

  const focus = () => inputEl.value?.focus()

  return { query, inputEl, matches, focus }
}
