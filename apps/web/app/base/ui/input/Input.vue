<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { input } from "@midoneui/styles/input.styles";

const modelValue = defineModel<string>()

const {
  class: className,
  type,
  ...props
} = defineProps<{
  class?: string;
  type?: string;
}>();
</script>

<template>
  <input :type="type" :class="cn(input, className)" v-model="modelValue" v-bind="{ ...props }" />
</template>
