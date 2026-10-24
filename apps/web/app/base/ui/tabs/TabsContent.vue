<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { tabsContent } from "@midoneui/styles/tabs.styles";
import type { Api, ContentProps } from "@zag-js/tabs";
import { inject } from "vue";
import { Slot } from "@/base/ui/slot";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<
  ContentProps & {
    class?: string;
    asChild?: boolean;
  }
>();

const api = inject<Api>("tabsApi");
</script>

<template>
  <Slot
    :class="cn(tabsContent, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getContentProps(props) }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
