import { computed } from 'vue'
import { useDarkModeStore } from '~/stores/dark-mode'

export const useDarkMode = () => {
    const darkModeStore = useDarkModeStore()

    const isDark = computed(() => darkModeStore.darkModeValue)

    const toggleDarkMode = () => {
        const next = !darkModeStore.darkModeValue
        darkModeStore.setDarkMode(next)
        document.documentElement.classList.toggle('dark', next)
    }

    return { isDark, toggleDarkMode }
}
