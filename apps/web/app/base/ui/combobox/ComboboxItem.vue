<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { comboboxItem } from "@midoneui/styles/combobox.styles";
import type { ItemProps } from "@zag-js/combobox";
import type { Api } from "@zag-js/combobox";
import { provide, inject } from "vue";
import { ComboboxItemIndicator } from ".";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<ItemProps & { class?: string; asChild?: boolean }>();

const api = inject<Api>("comboboxApi");

provide("comboboxItem", props);
</script>

<template>
  <Slot
    :class="cn(comboboxItem, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getItemProps(props) }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
      <ComboboxItemIndicator />
    </div>
  </Slot>
</template>
