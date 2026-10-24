<script setup lang="ts">
import { PopoverRoot, PopoverTrigger, PopoverPositioner, PopoverContent } from '~/base/ui/popover'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'

const { active = false } = defineProps<{
  /** Highlights the trigger and shows a dot when a filter is currently applied. */
  active?: boolean
}>()

const isOpen = ref(false)
</script>

<template>
  <PopoverRoot :open="isOpen" :onOpenChange="(d: { open: boolean }) => { isOpen = d.open }">
    <PopoverTrigger as-child>
      <Button
        type="button"
        look="outline"
        :variant="active ? 'primary' : 'ghost'"
        class="relative"
      >
        <Lucide icon="SlidersHorizontal" class="size-4" />
        Filter
        <span
          v-if="active"
          class="absolute -right-1 -top-1 size-2 rounded-full bg-primary"
        />
      </Button>
    </PopoverTrigger>
    <PopoverPositioner>
      <PopoverContent class="w-auto min-w-72 max-w-[90vw]">
        <div class="grid gap-3">
          <slot />
        </div>
        <div v-if="$slots.footer" class="mt-4 flex justify-end gap-2 border-t border-foreground/10 pt-3">
          <slot name="footer" />
        </div>
      </PopoverContent>
    </PopoverPositioner>
  </PopoverRoot>
</template>
