<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { Button } from "@/base/ui/button";
import { ChevronsUpDownIcon } from "lucide-vue-next";
import { comboboxTrigger } from "@midoneui/styles/combobox.styles";
import { ComboboxClearTrigger } from ".";
import type { Api } from "@zag-js/combobox";
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

const api = inject<Api>("comboboxApi");
</script>

<template>
  <Slot v-bind="{ ...props, ...$attrs, ...api?.getTriggerProps() }">
    <Button
      variant="ghost"
      v-if="!asChild"
      :class="cn(comboboxTrigger, className)"
    >
      <div>{{ api?.valueAsString || "Select Options..." }}</div>
      <ComboboxClearTrigger>Clear</ComboboxClearTrigger>
      <ChevronsUpDownIcon />
    </Button>
    <slot v-else />
  </Slot>
</template>
