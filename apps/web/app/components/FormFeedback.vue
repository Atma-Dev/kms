<script setup lang="ts">
const flashStore = useFlashStore()

defineProps({
  feedbackKey: {
    required: true,
    type: String,
  }
})
</script>

<template>
  <small class="text-xs text-danger" v-if="flashStore.isKeyErrors(feedbackKey)">{{flashStore.getErrorsByKey(feedbackKey)[0]}}</small>
</template>

<style scoped></style>
