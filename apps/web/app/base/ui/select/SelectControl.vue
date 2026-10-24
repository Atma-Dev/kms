<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { selectControl } from "@midoneui/styles/select.styles";
import { Slot } from "@/base/ui/slot";
import type { Api } from "@zag-js/select";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  ...props
} = defineProps<{
  class?: string;
  asChild?: boolean;
}>();

const api = inject<Api>("selectApi");
</script>

<template>
  <Slot
    :class="cn(selectControl, className)"
    v-bind="{ ...api?.getControlProps(), ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else><slot /></div>
  </Slot>
</template>
