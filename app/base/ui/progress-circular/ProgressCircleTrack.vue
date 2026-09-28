<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { progressCircleTrack } from "@midoneui/styles/progress-circular.styles";
import type { Api } from "@zag-js/progress";
import { inject } from "vue";

const { class: className, ...props } = defineProps<{
  class?: string;
}>();

const api = inject<Api>("progressApi");
</script>

<template>
  <circle
    :class="cn(progressCircleTrack, className)"
    v-bind="{ ...api?.getCircleTrackProps(), ...props, ...$attrs }"
  />
</template>
