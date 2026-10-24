<script setup lang="ts">
import { DialogCloseTrigger, DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import type M2MClient from '~/types/entities/m2m_client'

const emit = defineEmits<{ (e: 'confirm', id: string): void }>()

const isShowModal = ref(false)
const selected = ref<M2MClient | null>(null)

const handleModal = (isShow: boolean, client?: M2MClient) => {
  isShowModal.value = isShow
  selected.value = client ?? null
}

const onConfirm = () => {
  if (selected.value) emit('confirm', selected.value.id)
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }"
  >
    <DialogContent class="sm:max-w-md">
      <DialogTitle>{{ $t('account.m2mClients.revokeClientTitle') }}</DialogTitle>

      <div class="flex flex-col items-center gap-4 py-4 text-center">
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-danger/10">
          <Lucide icon="Ban" class="h-7 w-7 text-danger" />
        </div>
        <p class="text-sm text-foreground/70">
          {{ $t('account.m2mClients.revokeConfirm', { client: selected?.client }) }}
        </p>
      </div>

      <div class="flex justify-end gap-2 mt-2">
        <DialogCloseTrigger as-child @click="handleModal(false)">
          <Button look="outline" variant="ghost">{{ $t('common.cancel') }}</Button>
        </DialogCloseTrigger>
        <Button variant="danger" @click="onConfirm">
          <Lucide icon="Ban" class="mr-1.5 h-4 w-4" />
          {{ $t('account.m2mClients.revoke') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
