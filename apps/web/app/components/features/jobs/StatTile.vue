<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Lucide, type Icon } from '~/base/ui/lucide'

withDefaults(
  defineProps<{
    icon: Icon
    label: string
    value: string
    variant?: 'primary' | 'success' | 'warning' | 'danger' | 'secondary'
    hint?: string
  }>(),
  { variant: 'secondary', hint: '' },
)

const valueClass: Record<string, string> = {
  primary: 'text-primary',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger',
  secondary: '',
}
</script>

<template>
  <Box raised="single">
    <div class="flex items-center gap-1.5 text-xs uppercase opacity-70">
      <Lucide :icon="icon" class="size-3.5" />
      <span class="truncate">{{ label }}</span>
    </div>
    <div class="mt-1.5 text-2xl font-medium leading-8" :class="valueClass[variant]">
      {{ value }}
    </div>
    <div v-if="hint" class="mt-1 truncate text-xs opacity-40">{{ hint }}</div>
  </Box>
</template>
