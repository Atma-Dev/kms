<script lang="ts" setup>
import { Box } from '~/base/ui/box'

definePageMeta({
  title: 'home.title',
  pageSubTitle: 'home.subtitle',
  breadcrumb: ['Home'],
})

const { fullName } = storeToRefs(useAuthStore())
</script>

<template>
  <Box class="p-10 text-center">
    <div class="text-xl font-medium">{{ $t('home.greeting', { name: fullName }) }}</div>
    <div class="mt-2 text-foreground/70">{{ $t('home.message') }}</div>
  </Box>
</template>
