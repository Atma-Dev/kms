<script lang="ts" setup>
import {
  TooltipRoot,
  TooltipTrigger,
  TooltipPositioner,
  TooltipContent,
} from "@/base/ui/tooltip";
</script>

<template>
  <TooltipRoot>
    <TooltipTrigger>Hover Me</TooltipTrigger>
    <TooltipPositioner>
      <TooltipContent>I am a tooltip!</TooltipContent>
    </TooltipPositioner>
  </TooltipRoot>
</template>
