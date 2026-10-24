<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { paginationItem } from "@midoneui/styles/pagination.styles";
import { Slot } from "@/base/ui/slot";
import type { Api, ItemProps } from "@zag-js/pagination";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<
  ItemProps & {
    class?: string;
    asChild?: boolean;
  }
>();

const api = inject<Api>("paginationApi");
</script>

<template>
  <Slot
    :class="cn(paginationItem, className)"
    v-bind="{ ...api?.getItemProps(props), ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
