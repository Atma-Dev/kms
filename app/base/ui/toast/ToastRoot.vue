<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { toastRoot } from "@midoneui/styles/toast.styles";
import {
  boxVariants,
  type BoxVariants,
} from "@midoneui/styles/box.styles";
import { Slot } from "@/base/ui/slot";
import type { Api } from "@zag-js/toast";
import { inject } from "vue";

const {
  class: className,
  asChild = false,
  raised = "single",
  ...props
} = defineProps<
  BoxVariants & {
    class?: string;
    asChild?: boolean;
  }
>();

const api = inject<Api>("toastApi");
</script>

<template>
  <Slot
    :class="cn([boxVariants({ raised, className }), toastRoot, className])"
    v-bind="{ ...api?.getRootProps(), ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <div v-else>
      <span v-bind="{ ...api?.getGhostBeforeProps() }" />
      <div data-scope="toast" data-part="progressbar" />
      <slot />
      <span v-bind="{ ...api?.getGhostAfterProps() }" />
    </div>
  </Slot>
</template>
