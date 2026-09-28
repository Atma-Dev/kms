<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { fieldSet } from "@midoneui/styles/field.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <fieldset
    data-part="field-set"
    :class="cn(fieldSet, className)"
    v-bind="{ ...props }"
  >
    <slot />
  </fieldset>
</template>
