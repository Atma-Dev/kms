<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { comboboxInput } from "@midoneui/styles/combobox.styles";
import { Input } from "@/base/ui/input";
import type { Api } from "@zag-js/combobox";
import { inject } from "vue";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();

const api = inject<Api>("comboboxApi");
</script>

<template>
  <Input
    :class="cn(comboboxInput, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getInputProps() }"
  />
</template>
