<script setup lang="ts">
import { AlertRoot, AlertTitle, AlertDescription, AlertCloseTrigger } from '~/base/ui/alert'
import { Lucide } from '~/base/ui/lucide'
const flashStore = useFlashStore()
</script>

<template>
  <AlertRoot
    look="filled"
    :variant="flashStore.variant"
    v-if="flashStore.failedMessage || flashStore.successMessage"
  >
    <Lucide :icon="flashStore.icon" v-if="flashStore.icon" />
    <AlertTitle>{{ flashStore.title }}</AlertTitle>
    <AlertDescription v-if="flashStore.failedMessage">
      <div>{{ flashStore.failedMessage }}</div>
      <ul v-if="flashStore.errors" class="text-sm list-disc list-inside">
        <li
          v-for="(msg, index) in Object.values(flashStore.errors).flat()"
          :key="index"
          class="mb-1"
        >
          {{ msg }}
        </li>
      </ul>
    </AlertDescription>
    <AlertDescription v-if="flashStore.successMessage">
      <div>{{ flashStore.successMessage }}</div>
    </AlertDescription>
    <AlertCloseTrigger @click="flashStore.clearMessages()" />
  </AlertRoot>
</template>

<style scoped></style>
