<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'

const emit = defineEmits(["click-delete"])
</script>

<template>
  <a
    class="mr-3 flex items-center text-danger"
    href="#"
    @click.prevent="emit('click-delete')"
  >
    <Lucide class="mr-1 h-4 w-4" icon="Trash" />
    {{ $t('common.delete') }}
  </a>
</template>

<style scoped></style>
