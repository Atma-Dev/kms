<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import {
  boxVariants,
  type BoxVariants,
} from "@midoneui/styles/box.styles";
import { Slot } from "@/base/ui/slot";

const {
  class: className,
  asChild = false,
  raised,
  ...props
} = defineProps<
  BoxVariants & {
    class?: string;
    asChild?: boolean;
  }
>();
</script>

<template>
  <Slot
    :class="cn(boxVariants({ raised, className }), className)"
    v-bind="{ ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
