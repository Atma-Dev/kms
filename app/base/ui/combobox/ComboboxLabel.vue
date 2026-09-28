<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { comboboxLabel } from "@midoneui/styles/combobox.styles";
import { Label } from "@/base/ui/label";
import type { Api } from "@zag-js/combobox";
import { inject } from "vue";

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
  <Slot v-bind="{ ...props, ...$attrs, ...api?.getLabelProps() }">
    <slot v-if="asChild" />
    <Label v-else :class="cn(comboboxLabel, className)"><slot /></Label>
  </Slot>
</template>
