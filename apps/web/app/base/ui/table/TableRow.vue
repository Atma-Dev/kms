<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { tableRow } from "@midoneui/styles/table.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <tr :class="cn(tableRow, className)" v-bind="{ ...props }">
    <slot />
  </tr>
</template>
