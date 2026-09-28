<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { fieldDescription } from "@midoneui/styles/field.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <p
    data-part="field-description"
    :class="cn(fieldDescription, className)"
    v-bind="{ ...props }"
  >
    <slot />
  </p>
</template>
