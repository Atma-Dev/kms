<script lang="ts" setup>
import type { Api } from "@zag-js/combobox";
import { inject } from "vue";
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { comboboxControl } from "@midoneui/styles/combobox.styles";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("comboboxApi");
</script>

<template>
  <Slot
    :class="cn(comboboxControl, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getLabelProps() }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
