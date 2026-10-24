<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { label } from "@midoneui/styles/label.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <label :class="cn(label, className)" v-bind="{ ...props }">
    <slot />
  </label>
</template>
