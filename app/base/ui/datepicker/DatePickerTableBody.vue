<script lang="ts" setup>
import { Slot } from "@/base/ui/slot";
import { cn } from "@midoneui/utils/cn";
import { datePickerTableBody } from "@midoneui/styles/datepicker.styles";
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
    :class="cn(datePickerTableBody, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getTableHeadProps() }"
  >
    <slot v-if="asChild" />
    <tbody v-else>
      <slot />
    </tbody>
  </Slot>
</template>
