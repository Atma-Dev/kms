<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { toastCloseTrigger } from "@midoneui/styles/toast.styles";
import {
  buttonVariants,
  type ButtonVariants,
} from "@midoneui/styles/button.styles";
import { Slot } from "@/base/ui/slot";
import { Button } from "@/base/ui/button";
import { X } from "lucide-vue-next";
import type { Api } from "@zag-js/toast";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  look = "outline",
  variant = "secondary",
  size,
  ...props
} = defineProps<
  ButtonVariants & {
    class?: string;
    asChild?: boolean;
  }
>();

const api = inject<Api>("toastApi");
</script>

<template>
  <Slot v-bind="{ ...api?.getCloseTriggerProps(), ...props, ...$attrs }">
    <Button
      variant="ghost"
      v-if="!$slots.default"
      :class="cn(toastCloseTrigger, className)"
      v-bind="{ ...props }"
    >
      <X class="size-4" />
    </Button>
    <template v-else>
      <slot v-if="asChild" />
      <Button
        v-else
        :class="
          cn(buttonVariants({ look, variant, size, className }), className)
        "
      >
        <slot />
      </Button>
    </template>
  </Slot>
</template>
