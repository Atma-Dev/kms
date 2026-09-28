<script lang="ts" setup generic="TType extends ChartType">
import ChartJs from "chart.js/auto";
import { ref, onMounted, onUnmounted } from "vue";
import { cn } from "@midoneui/utils/cn";
import { chart } from "@midoneui/styles/chart.styles";
import type { ChartType, ChartConfiguration } from "chart.js";

const {
  class: className,
  config,
  getRef,
  ...props
} = defineProps<{
  class?: string;
  config: ChartConfiguration<TType>;
  getRef?: (chart: ChartJs<TType>) => void;
}>();

const chartRef = ref<
  | (HTMLCanvasElement & {
      instance?: ChartJs<TType>;
    })
  | null
>(null);

onMounted(() => {
  if (chartRef.value && !chartRef.value.instance) {
    chartRef.value.instance = new ChartJs(chartRef.value, config);
    getRef?.(chartRef.value.instance);
  }
});

// Chart.js's responsive mode measures the canvas's parent to size the
// canvas, then applies that size back to the canvas — putting the height
// class directly on the canvas (no sized wrapper) turns that into a
// feedback loop that grows the chart taller on every tick.
onUnmounted(() => {
  chartRef.value?.instance?.destroy();
});
</script>

<template>
  <div :class="cn(chart, 'relative', className)">
    <canvas class="absolute inset-0 size-full" ref="chartRef" v-bind="props" />
  </div>
</template>
