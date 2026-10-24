<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { selectTrigger } from "@midoneui/styles/select.styles";
import { Slot } from "@/base/ui/slot";
import { Button } from "@/base/ui/button";
import { SelectClearTrigger, SelectIndicator } from ".";
import type { Api } from "@zag-js/select";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("selectApi");
</script>

<template>
  <Slot v-bind="{ ...api?.getTriggerProps(), ...props, ...$attrs }">
    <Button
      variant="ghost"
      v-if="!asChild"
      :class="cn(selectTrigger, className)"
    >
      <slot />
      <SelectClearTrigger>Clear</SelectClearTrigger>
      <SelectIndicator />
    </Button>
    <slot v-else />
  </Slot>
</template>
