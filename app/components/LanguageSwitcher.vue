<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Lucide } from '~/base/ui/lucide'

const { locale, locales, setLocale } = useI18n()

const localeOptions = computed(() => locales.value as { code: string; name: string }[])

const FLAGS: Record<string, string> = { id: '🇮🇩', en: '🇬🇧' }

const switchLocale = async (code: string) => {
  await setLocale(code)
  localStorage.setItem('locale', code)
}
</script>

<template>
  <div class="group/language relative flex h-9 items-center">
    <button type="button" class="flex size-9 cursor-pointer items-center justify-center rounded-full hover:bg-foreground/5">
      <span class="text-lg leading-none">{{ FLAGS[locale] }}</span>
    </button>
    <div
      class="invisible absolute right-0 top-full z-50 mt-2 origin-top-right scale-95 opacity-0 transition-all duration-200 delay-0 group-hover/language:visible group-hover/language:scale-100 group-hover/language:opacity-100 group-hover/language:delay-200"
    >
      <Box class="before:shadow-foreground/5 z-50 flex w-40 flex-col gap-1 p-2 before:rounded-2xl before:shadow-xl before:backdrop-blur after:rounded-2xl">
        <button
          v-for="opt in localeOptions"
          :key="opt.code"
          type="button"
          class="hover:bg-foreground/5 flex items-center gap-2 rounded-xl px-3 py-2 text-left text-sm"
          :class="{ 'text-primary font-medium': locale === opt.code }"
          @click="switchLocale(opt.code)"
        >
          <span class="text-base leading-none">{{ FLAGS[opt.code] }}</span>
          <span class="flex-1">{{ opt.code === 'id' ? $t('nav.languageIndonesian') : $t('nav.languageEnglish') }}</span>
          <Lucide v-if="locale === opt.code" icon="Check" class="!size-3.5" />
        </button>
      </Box>
    </div>
  </div>
</template>
