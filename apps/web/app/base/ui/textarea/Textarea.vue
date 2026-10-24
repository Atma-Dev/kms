<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { textarea } from "@midoneui/styles/textarea.styles";

const modelValue = defineModel<string>()

const { class: className, ...props } = defineProps<{
  class?: string;
}>();
</script>

<template>
  <textarea :class="cn(textarea, className)" v-model="modelValue" v-bind="{ ...props }" />
</template>
