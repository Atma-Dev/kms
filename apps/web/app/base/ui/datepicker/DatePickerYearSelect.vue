<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/base/ui/native-select";
import { datePickerYearSelect } from "@midoneui/styles/datepicker.styles";
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
  <NativeSelect
    :class="cn(datePickerYearSelect, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getYearSelectProps() }"
  >
    <NativeSelectOption
      v-for="(year, i) in api?.getYears()"
      :key="i"
      :value="year.value"
    >
      {{ year.label }}
    </NativeSelectOption>
  </NativeSelect>
</template>
