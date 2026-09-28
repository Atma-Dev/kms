<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { NativeSelectOptGroup } from "@midoneui/styles/native-select.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <optgroup
    :class="cn(NativeSelectOptGroup, className)"
    v-bind="{ ...props, ...$attrs }"
  >
    <slot />
  </optgroup>
</template>
