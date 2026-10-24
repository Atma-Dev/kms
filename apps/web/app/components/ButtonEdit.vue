<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'
const emit = defineEmits(['click-edit'])
</script>

<template>
  <a class="mr-3 flex items-center" href="#" @click.prevent="emit('click-edit')">
    <Lucide class="mr-1 h-4 w-4" icon="CheckSquare" />
    {{ $t('common.edit') }}
  </a>
</template>

<style scoped></style>
