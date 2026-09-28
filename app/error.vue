<script lang="ts" setup>
import type { NuxtError } from '#app'
import errorIllustrationUrl from '~/assets/images/error-illustration.svg'
import { Button } from '~/base/ui/button'

// The template's error-page.vue. Nuxt renders this instead of the app for an
// unmatched route (404), for any error thrown while rendering (500), and for a
// 403 raised via showError() when a page's data comes back forbidden. It sits
// outside the layout system — no side menu on a page that failed — and the three
// statuses differ only in the copy, so one screen covers all of them.
const props = defineProps<{ error: NuxtError }>()

const { t } = useI18n()

const COPY_KEY: Record<number, string> = { 403: 'forbidden', 404: 'notFound' }

const copy = computed(() => {
  const key = COPY_KEY[props.error.statusCode ?? 0] ?? 'serverError'
  return {
    title: t(`common.${key}.title`),
    message: t(`common.${key}.message`),
    action: t(`common.${key}.back`),
  }
})

// clearError drops the error state before navigating; without it the error page
// stays mounted over the route it just moved to.
const goHome = () => clearError({ redirect: '/home' })
</script>

<template>
  <div
    :class="[
      'relative py-2',
      'before:bg-primary dark:before:bg-foreground/[.01] before:fixed before:inset-0 before:bg-noise',
      'after:bg-accent after:bg-contain after:fixed after:inset-0 after:blur-xl dark:after:opacity-20',
    ]"
  >
    <div class="container relative z-10 mx-auto px-4">
      <div
        class="flex h-screen flex-col items-center justify-center text-center lg:flex-row lg:text-left"
      >
        <div class="lg:mr-20">
          <img
            class="h-auto w-full max-w-[280px] sm:max-w-[450px]"
            :src="errorIllustrationUrl"
            alt=""
          />
        </div>
        <div class="mt-10 text-white lg:mt-0">
          <div class="text-6xl sm:text-8xl lg:text-9xl [text-shadow:_7px_7px_--alpha(var(--color-white)_/_20%)]">
            {{ error.statusCode || 500 }}
          </div>
          <div class="mt-8 text-xl font-medium lg:text-2xl">{{ copy.title }}</div>
          <div class="mt-3 text-base opacity-70">{{ copy.message }}</div>
          <Button
            class="box mt-10 border border-white bg-transparent px-7 py-6 text-white"
            variant="ghost"
            @click="goHome"
          >
            {{ copy.action }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
