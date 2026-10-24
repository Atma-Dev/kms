<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { dialogDescription } from "@midoneui/styles/dialog.styles";
import type { Api } from "@zag-js/dialog";
import { inject } from "vue";
import { Slot } from "@/base/ui/slot";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("dialogApi");
</script>

<template>
  <Slot
    :class="cn(dialogDescription, className)"
    v-bind="{ ...props, ...$attrs, ...api?.getDescriptionProps() }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <slot />
    </div>
  </Slot>
</template>
