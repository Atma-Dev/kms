<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { datePickerClearTrigger } from "@midoneui/styles/datepicker.styles";
import { Button } from "@/base/ui/button";
import type { Api } from "@zag-js/date-picker";
import { inject } from "vue";
import { X } from "lucide-vue-next";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("datepickerApi");
</script>

<template>
  <Slot v-bind="{ ...props, ...$attrs, ...api?.getClearTriggerProps() }">
    <Button
      variant="ghost"
      v-if="!asChild"
      :class="cn(datePickerClearTrigger, className)"
    >
      <X v-if="!$slots.default" />
      <slot v-else />
    </Button>
    <slot v-else />
  </Slot>
</template>
