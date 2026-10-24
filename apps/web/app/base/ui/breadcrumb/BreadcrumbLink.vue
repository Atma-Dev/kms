<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { breadcrumbLink } from "@midoneui/styles/breadcrumb.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <a :class="cn(className, breadcrumbLink)" v-bind="{ ...props, ...$attrs }">
    <slot />
  </a>
</template>
