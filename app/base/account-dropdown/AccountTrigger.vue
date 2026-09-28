<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'

interface Props {
  class?: string
  innerClass?: any
  avatarClass?: string
  textClass?: string
}

const props = defineProps<Props>()
// storeToRefs — a plain destructure of a setup store yields the value, not the
// ref, so the header would keep showing stale data after `fetchMe()`.
const { user, fullName } = storeToRefs(useAuthStore())
</script>

<template>
  <div :class="['side-menu__account group/profile transition-[width]', props.class]">
    <div :class="['flex cursor-pointer items-center transition', props.innerClass]">
      <div
        :class="[
          'relative flex-none overflow-hidden rounded-full border-4 flex items-center justify-center bg-primary/20',
          props.avatarClass ?? 'h-10 w-10 border-background/20 dark:border-foreground/20',
        ]"
      >
        <span class="text-sm font-semibold text-primary uppercase">
          {{ user?.first_name?.[0] }}{{ user?.last_name?.[0] }}
        </span>
      </div>
      <div
        class="ms-3 flex w-full items-center overflow-hidden transition-opacity group-[.side-menu--collapsed.side-menu--on-hover]:ms-3 group-[.side-menu--collapsed.side-menu--on-hover]:w-full group-[.side-menu--collapsed.side-menu--on-hover]:opacity-100 xl:group-[.side-menu--collapsed]:ms-0 xl:group-[.side-menu--collapsed]:w-0 xl:group-[.side-menu--collapsed]:opacity-0"
      >
        <div :class="['w-28', props.textClass]">
          <div class="w-full truncate font-medium">{{ fullName }}</div>
          <div class="w-full truncate text-xs opacity-60">{{ user?.email }}</div>
        </div>
        <Lucide class="me-4 ms-auto opacity-50 flex-none" icon="MoveRight" />
      </div>
    </div>
    <slot />
  </div>
</template>
