import { ref, watch } from 'vue'

// Dark is the default; only an explicit stored choice switches to light.
// Read at module load: the immediate watcher below would otherwise overwrite the stored value first.
const isDark = ref(localStorage.getItem('theme') !== 'light')

export function useTheme() {
  const toggleTheme = () => {
    isDark.value = !isDark.value
  }

  const setTheme = (dark: boolean) => {
    isDark.value = dark
  }

  watch(isDark, (dark) => {
    localStorage.setItem('theme', dark ? 'dark' : 'light')
    if (dark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }, { immediate: true })

  return {
    isDark,
    toggleTheme,
    setTheme,
  }
}
