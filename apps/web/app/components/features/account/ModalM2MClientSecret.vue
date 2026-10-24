<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'

const client = ref('')
const secret = ref('')
const isShowModal = ref(false)
const copied = ref(false)

const handleModal = (isShow: boolean, payload?: { client: string; secret: string }) => {
  isShowModal.value = isShow
  copied.value = false
  if (payload) {
    client.value = payload.client
    secret.value = payload.secret
  }
}

const copy = () => {
  navigator.clipboard.writeText(secret.value).then(() => {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  })
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(d: { open: boolean }) => { if (!d.open) handleModal(false) }"
  >
    <DialogContent class="sm:max-w-md">
      <DialogTitle>{{ $t('account.m2mClients.clientSecretTitle') }}</DialogTitle>

      <div class="flex flex-col gap-4 mt-4">
        <div class="flex items-start gap-2 rounded-lg bg-warning/10 p-3 text-warning text-sm">
          <Lucide icon="TriangleAlert" class="h-4 w-4 shrink-0 mt-0.5" />
          <span>
            {{ $t('account.m2mClients.secretWarning', { client }) }}
          </span>
        </div>

        <div class="flex items-center gap-2">
          <code class="flex-1 truncate rounded-lg border border-foreground/10 bg-foreground/5 px-3 py-2 text-xs">
            {{ secret }}
          </code>
          <Button look="outline" variant="ghost" @click="copy">
            <Lucide :icon="copied ? 'Check' : 'Copy'" />
            {{ copied ? $t('account.m2mClients.copied') : $t('account.m2mClients.copy') }}
          </Button>
        </div>
      </div>

      <div class="flex justify-end mt-7">
        <Button variant="primary" @click="handleModal(false)">{{ $t('common.close') }}</Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
