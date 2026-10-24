<script setup lang="ts">
import { TableHead } from '~/base/ui/table'
import { type Icon, Lucide } from '~/base/ui/lucide'

import type { PropType } from 'vue'
defineProps({
  icon: {
    required: false,
    type: String as PropType<Icon>,
  },
})
</script>

<template>
  <TableHead
    class="border-0 bg-foreground/5 text-foreground/50 text-[11px] font-semibold uppercase tracking-wider py-3"
  >
    <div class="flex items-center gap-1.5">
      <Lucide v-if="icon" :icon="icon" class="w-3.5 h-3.5" />
      <slot></slot>
    </div>
  </TableHead>
</template>

<style scoped></style>
