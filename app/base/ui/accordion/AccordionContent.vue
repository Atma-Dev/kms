<script lang="ts" setup>
import { accordionContent } from "@midoneui/styles/accordion.styles";
import { cn } from "@midoneui/utils/cn";
import { inject } from "vue";
import { Slot } from "@/base/ui/slot";
import type { Api, ItemProps } from "@zag-js/accordion";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("accordionApi");
const item = inject<ItemProps>("accordionItem");
</script>

<template>
  <Slot
    :class="cn([className, accordionContent])"
    v-bind="{ ...props, ...$attrs, ...api?.getItemContentProps(item!) }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
