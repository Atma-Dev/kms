<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { Label } from "@/base/ui/label";
import { fieldLabel } from "@midoneui/styles/field.styles";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <Label
    data-part="field-label"
    :class="cn(fieldLabel, className)"
    v-bind="{ ...props }"
  >
    <slot />
  </Label>
</template>
