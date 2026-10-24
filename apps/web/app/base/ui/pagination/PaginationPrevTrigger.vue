<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { paginationPrevTrigger } from "@midoneui/styles/pagination.styles";
import { Slot } from "@/base/ui/slot";
import type { Api } from "@zag-js/pagination";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("paginationApi");
</script>

<template>
  <Slot
    :class="cn(paginationPrevTrigger, className)"
    v-bind="{ ...api?.getPrevTriggerProps(), ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
