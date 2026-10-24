<script setup lang="ts">
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'

const dataModel = defineModel<string>()
</script>

<template>
  <div class="mt-3 w-full sm:ml-auto sm:mt-0 sm:w-auto md:ml-0">
    <div class="relative w-56">
      <Input class="w-56 pr-10" type="text" v-model="dataModel" :placeholder="$t('common.search')" />
      <Lucide class="absolute inset-y-0 right-0 my-auto mr-3 h-4 w-4" icon="Search" />
    </div>
  </div>
</template>

<style scoped></style>
