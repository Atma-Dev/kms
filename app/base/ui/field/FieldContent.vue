<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { fieldContent } from "@midoneui/styles/field.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <div
    data-part="field-content"
    :class="cn(fieldContent, className)"
    v-bind="{ ...props }"
  >
    <slot />
  </div>
</template>
