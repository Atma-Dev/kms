<script setup lang="ts">
import { useThemeStore, getTheme, themes, type Themes } from '~/stores/theme'
import { ThemeSwitcher } from '~/base/theme-switcher'
import { useRoute } from 'vue-router'
import { onMounted, computed } from 'vue'

const route = useRoute()
const Component = computed(() => getTheme(themeStore.theme).component)

const themeStore = useThemeStore()
const switchTheme = (theme: Themes['name']) => {
  useThemeStore().setTheme(theme)
}

onMounted(() => {
  const theme = route.query.theme as Themes['name']
  if (route.query.theme !== undefined && themes.map((theme) => theme.name).includes(theme)) {
    switchTheme(theme)
  }
})

const { t } = useI18n()

useHead({
  title: computed(() => (route.meta.title ? t(route.meta.title as string) : 'Atma KMS')),
  titleTemplate: (title) => (title && title !== 'Atma KMS' ? `${title} · Atma KMS` : 'Atma KMS'),
})
</script>

<template>
  <div>
    <ThemeSwitcher />
    <Component>
      <SimpleAlert />

      <h2 class="text-lg font-medium" v-if="route.meta.title">{{ t(route.meta.title as string) }}</h2>
      <p class="text-xs" v-if="route.meta.pageSubTitle">{{ t(route.meta.pageSubTitle as string) }}</p>

      <div class="mt-5">
        <slot />
      </div>
    </Component>
  </div>
</template>
