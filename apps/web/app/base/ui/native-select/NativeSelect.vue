<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { input } from "@midoneui/styles/input.styles";
import { nativeSelect } from "@midoneui/styles/native-select.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <select
    :class="cn(input, nativeSelect, className)"
    v-bind="{ ...props, ...$attrs }"
  >
    <slot />
  </select>
</template>
