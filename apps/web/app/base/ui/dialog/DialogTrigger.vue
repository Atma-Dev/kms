<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { dialogTrigger } from "@midoneui/styles/dialog.styles";
import { Button } from "@/base/ui/button";
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
  <Slot v-bind="{ ...props, ...$attrs, ...api?.getTriggerProps() }">
    <Button
      variant="secondary"
      look="outline"
      v-if="!asChild"
      :class="cn(dialogTrigger, className)"
    >
      <slot />
    </Button>
    <slot v-else />
  </Slot>
</template>
