<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { datePickerTable } from "@midoneui/styles/datepicker.styles";
import type { Api } from "@zag-js/date-picker";
import { inject } from "vue";

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
  <Slot
    :class="cn(datePickerTable, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getTableProps() }"
  >
    <slot v-if="asChild" />
    <table v-else>
      <slot />
    </table>
  </Slot>
</template>
