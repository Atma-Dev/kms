<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import type { Api } from "@zag-js/checkbox";
import { cn } from "@midoneui/utils/cn";
import { checkboxIndicator } from "@midoneui/styles/checkbox.styles";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  asChild?: boolean;
  class?: string;
}>();

const api = inject<Api>("checkboxApi");
</script>

<template>
  <Slot
    :class="cn(checkboxIndicator, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getIndicatorProps() }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
