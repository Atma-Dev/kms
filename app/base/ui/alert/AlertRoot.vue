<script lang="ts" setup>
import { cn } from "@midoneui/utils/cn";
import {
  alertRootVariants,
  type AlertRootVariants,
} from "@midoneui/styles/alert.styles";
import { Presence } from "@/base/ui/presence";
import { ref, provide } from "vue";

const {
  class: className,
  look,
  variant,
  ...rest
} = defineProps<AlertRootVariants & { class?: string }>();

const present = ref(true);
const setPresent = (value: boolean) => {
  present.value = value;
};

provide("alertPresent", { present, setPresent });
</script>

<template>
  <Presence
    :class="
      cn(
        alertRootVariants({
          look,
          variant,
        }),
        className
      )
    "
    v-bind="rest"
    :present="present"
  >
    <slot />
  </Presence>
</template>
