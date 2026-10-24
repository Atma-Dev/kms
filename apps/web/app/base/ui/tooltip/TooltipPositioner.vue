<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { tooltipPositioner } from "@midoneui/styles/tooltip.styles";
import { Slot } from "@/base/ui/slot";
import type { Api } from "@zag-js/tooltip";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("tooltipApi");
</script>

<template>
  <Teleport to="body">
    <Slot
      :class="cn(tooltipPositioner, className)"
      v-bind="{ ...api?.getPositionerProps(), ...props, ...$attrs }"
    >
      <slot v-if="asChild" />
      <div v-else>
        <slot />
      </div>
    </Slot>
  </Teleport>
</template>
