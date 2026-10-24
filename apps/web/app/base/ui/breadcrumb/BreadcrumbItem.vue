<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { breadcrumbItem } from "@midoneui/styles/breadcrumb.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <li :class="cn(className, breadcrumbItem)" v-bind="{ ...props, ...$attrs }">
    <slot />
  </li>
</template>
