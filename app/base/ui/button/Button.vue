<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import {
  buttonVariants,
  type ButtonVariants,
} from "@midoneui/styles/button.styles";
import { Slot } from "@/base/ui/slot";

const {
  class: className,
  look,
  variant,
  size,
  asChild = false,
  ...props
} = defineProps<
  ButtonVariants & {
    class?: string;
    asChild?: boolean;
  }
>();
</script>

<template>
  <Slot
    v-bind="{ ...props, ...$attrs }"
    :class="cn(buttonVariants({ look, variant, size, className }), className)"
  >
    <slot v-if="asChild" />
    <button v-else>
      <slot />
    </button>
  </Slot>
</template>
