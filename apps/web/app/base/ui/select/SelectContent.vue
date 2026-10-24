<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { selectContent } from "@midoneui/styles/select.styles";
import { Slot } from "@/base/ui/slot";
import { Box } from "@/base/ui/box";
import { SelectPositioner } from ".";
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
  <Teleport to="body">
    <SelectPositioner>
      <Slot v-bind="{ ...api?.getContentProps(), ...props, ...$attrs }">
        <slot v-if="asChild" />
        <Box v-else raised="single" :class="cn(selectContent, className)">
          <div><slot /></div>
        </Box>
      </Slot>
    </SelectPositioner>
  </Teleport>
</template>
