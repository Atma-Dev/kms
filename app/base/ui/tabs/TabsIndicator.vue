<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { tabsIndicator } from "@midoneui/styles/tabs.styles";
import type { Api } from "@zag-js/tabs";
import { inject } from "vue";
import { Slot } from "@/base/ui/slot";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("tabsApi");
</script>

<template>
  <Slot
    :class="cn(tabsIndicator, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getIndicatorProps() }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
