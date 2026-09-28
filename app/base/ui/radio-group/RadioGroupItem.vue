<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import { radioGroupItem } from "@midoneui/styles/radio-group.styles";
import { Slot } from "@/base/ui/slot";
import { RadioGroupItemHiddenInput } from ".";
import type { Api, ItemProps } from "@zag-js/radio-group";
import { provide, inject } from "vue";

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

const api = inject<Api>("radioGroupApi");

provide("radioGroupItem", props);
</script>

<template>
  <Slot
    :class="cn(radioGroupItem, className)"
    v-bind="{ ...api?.getItemProps(props), ...props, ...$attrs }"
  >
    <slot v-if="asChild" />
    <label v-else>
      <slot />
      <RadioGroupItemHiddenInput />
    </label>
  </Slot>
</template>
