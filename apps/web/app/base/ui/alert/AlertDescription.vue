<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { Slot } from "@/base/ui/slot";
import { alertDescription } from "@midoneui/styles/alert.styles";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();
</script>

<template>
  <Slot
    :class="cn([className, alertDescription])"
    v-bind="{ ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
