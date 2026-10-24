<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { type M2MClientCreated, useM2MClientService } from '~/services/M2MClientService'
import type M2MClient from '~/types/entities/m2m_client'
import type { ClientSidePaginationProps } from '~/types/response'
import ModalFormM2MClient from '~/components/features/account/ModalFormM2MClient.vue'
import ModalM2MClientSecret from '~/components/features/account/ModalM2MClientSecret.vue'
import ModalRevokeM2MClient from '~/components/features/account/ModalRevokeM2MClient.vue'
import ModalRegenerateM2MClientSecret from '~/components/features/account/ModalRegenerateM2MClientSecret.vue'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'account.m2mClients.title',
  pageSubTitle: 'account.m2mClients.subtitle',
  breadcrumb: ['M2M Clients'],
})

const authStore = useAuthStore()
const canStore = computed(() => authStore.hasPermission(Permission.MANAGEMENT_M2M_CLIENTS_STORE))
const canUpdate = computed(() => authStore.hasPermission(Permission.MANAGEMENT_M2M_CLIENTS_UPDATE))
const canDestroy = computed(() => authStore.hasPermission(Permission.MANAGEMENT_M2M_CLIENTS_DESTROY))

const { clients, loading, getAll, revokeById, regenerateSecret } = useM2MClientService()

// ponytail: one shot of the API cap (100). Go back to useServerList if this outgrows it.
const refresh = () => getAll()
onMounted(refresh)

const searchKey = ref('')
const filteredData = computed((): M2MClient[] => {
  const key = searchKey.value.toLowerCase()
  return key === '' ? clients.value : clients.value.filter((c) => c.client.toLowerCase().includes(key))
})

const modalFormRef = useTemplateRef<InstanceType<typeof ModalFormM2MClient> | null>('modalFormRef')
const modalSecretRef = useTemplateRef<InstanceType<typeof ModalM2MClientSecret> | null>('modalSecretRef')
const modalRevokeRef = useTemplateRef<InstanceType<typeof ModalRevokeM2MClient> | null>('modalRevokeRef')
const modalRegenerateRef = useTemplateRef<InstanceType<typeof ModalRegenerateM2MClientSecret> | null>(
  'modalRegenerateRef',
)

const onCreated = (created: M2MClientCreated) => {
  refresh()
  modalSecretRef.value?.handleModal(true, { client: created.client, secret: created.secret })
}

const askRevoke = (client: M2MClient) => modalRevokeRef.value?.handleModal(true, client)

const onRevokeConfirmed = async (id: string) => {
  if (await revokeById(id)) refresh()
  modalRevokeRef.value?.handleModal(false)
}

// X-Client-Id is client.id (the row's uuid), not client.client (the display
// name) — see middleware.loadM2MClient on the API. Copy-to-clipboard here so
// no one has to select-and-copy a uuid by hand and risk a transcription miss.
const copiedClientId = ref<string | null>(null)
const copyClientId = (id: string) => {
  navigator.clipboard.writeText(id).then(() => {
    copiedClientId.value = id
    setTimeout(() => {
      if (copiedClientId.value === id) copiedClientId.value = null
    }, 1500)
  })
}

const askRegenerate = (client: M2MClient) => modalRegenerateRef.value?.handleModal(true, client)

const onRegenerateConfirmed = async (id: string) => {
  const regenerated = await regenerateSecret(id)
  modalRegenerateRef.value?.handleModal(false)
  if (regenerated) {
    refresh()
    modalSecretRef.value?.handleModal(true, { client: regenerated.client, secret: regenerated.secret })
  }
}
</script>

<template>
  <Box class="overflow-auto lg:overflow-visible">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <Button v-if="canStore" @click="modalFormRef?.handleModal(true)">
        <Lucide icon="Plus" />
        {{ $t('account.m2mClients.addClient') }}
      </Button>
      <ClientSideFilter v-model="searchKey" />
    </div>

    <div v-if="loading" class="flex items-center justify-center gap-3 py-16 text-foreground/50">
      <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
      </svg>
      <span class="text-sm">{{ $t('common.loading') }}</span>
    </div>

    <ClientSidePagination
      v-else
      :data="filteredData"
      v-slot="slotProps: ClientSidePaginationProps<M2MClient>"
    >
      <TableHeader>
        <TableRow>
          <TH icon="Hash">{{ $t('common.no') }}</TH>
          <TH icon="KeySquare">{{ $t('account.m2mClients.columnClient') }}</TH>
          <TH icon="ShieldCheck">{{ $t('account.m2mClients.columnScope') }}</TH>
          <TH icon="CircleCheck">{{ $t('common.status') }}</TH>
          <TH icon="Clock">{{ $t('common.createdAt') }}</TH>
          <TH icon="Settings">{{ $t('common.actions') }}</TH>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="(client, index) in slotProps.data" :key="client.id">
          <TD>{{ slotProps.start + index }}</TD>
          <TD>
            <p class="font-medium">{{ client.client }}</p>
            <div class="flex items-center gap-1 mt-1">
              <span class="text-[10px] uppercase tracking-wide text-foreground/40">
                {{ $t('account.m2mClients.clientIdLabel') }}
              </span>
              <code class="text-[10px] text-foreground/60 bg-foreground/5 rounded px-1.5 py-0.5">
                {{ client.id }}
              </code>
              <button
                type="button"
                class="text-foreground/40 hover:text-foreground/70"
                :title="$t('account.m2mClients.copy')"
                @click="copyClientId(client.id)"
              >
                <Lucide :icon="copiedClientId === client.id ? 'Check' : 'Copy'" class="w-3 h-3" />
              </button>
            </div>
          </TD>
          <TD>
            <div class="flex flex-wrap gap-1">
              <Badge v-for="scope in client.scope" :key="scope" variant="secondary" look="outline" class="text-xs">
                {{ scope }}
              </Badge>
            </div>
          </TD>
          <TD>
            <span
              v-if="!client.revoked_at"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-success/10 text-success text-xs font-medium"
            >
              <Lucide icon="CircleCheck" class="w-3 h-3" />
              {{ $t('account.m2mClients.active') }}
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-foreground/10 text-foreground/50 text-xs font-medium"
            >
              <Lucide icon="Ban" class="w-3 h-3" />
              {{ $t('account.m2mClients.revoked') }}
            </span>
          </TD>
          <TD>{{ client.created_at }}</TD>
          <TD>
            <div class="flex items-center gap-1">
              <ButtonEdit v-if="canUpdate" @click-edit="modalFormRef?.handleModal(true, client)" />
              <a
                v-if="!client.revoked_at && canUpdate"
                class="mr-3 flex items-center text-warning"
                href="#"
                @click.prevent="askRegenerate(client)"
              >
                <Lucide class="mr-1 h-4 w-4" icon="RefreshCw" />
                {{ $t('account.m2mClients.regenerateSecret') }}
              </a>
              <a
                v-if="!client.revoked_at && canDestroy"
                class="mr-3 flex items-center text-danger"
                href="#"
                @click.prevent="askRevoke(client)"
              >
                <Lucide class="mr-1 h-4 w-4" icon="Ban" />
                {{ $t('account.m2mClients.revoke') }}
              </a>
            </div>
          </TD>
        </TableRow>
      </TableBody>
    </ClientSidePagination>
  </Box>

  <ModalFormM2MClient ref="modalFormRef" @created="onCreated" @updated="refresh()" />
  <ModalM2MClientSecret ref="modalSecretRef" />
  <ModalRevokeM2MClient ref="modalRevokeRef" @confirm="onRevokeConfirmed" />
  <ModalRegenerateM2MClientSecret ref="modalRegenerateRef" @confirm="onRegenerateConfirmed" />
</template>
