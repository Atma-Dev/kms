<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { progressRange } from "@midoneui/styles/progress-linear.styles";
import type { Api } from "@zag-js/progress";
import { inject } from "vue";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();

const api = inject<Api>("progressApi");
</script>

<template>
  <div
    :class="cn(progressRange, className)"
    v-bind="{ ...api?.getRangeProps(), ...props, ...$attrs }"
  />
</template>
