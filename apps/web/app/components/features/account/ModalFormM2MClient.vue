<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import type M2MClient from '~/types/entities/m2m_client'
import {
  type M2MClientCreated,
  type M2MClientRequest,
  useM2MClientService,
} from '~/services/M2MClientService'

const emit = defineEmits<{
  (e: 'created', client: M2MClientCreated): void
  (e: 'updated'): void
}>()

const { addNew, updateById, submitting } = useM2MClientService()
const flashStore = useFlashStore()

const isShowModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)

// Fixed set — mirrors m2mclient.Scope* on the API. "*" grants every scope, so
// it is offered as its own option rather than requiring both listed.
const { t } = useI18n()
const SCOPE_OPTIONS = computed(() => [
  { id: '*', label: t('account.m2mClients.scopeAll') },
  { id: 'invoices', label: t('account.m2mClients.scopeInvoices') },
  { id: 'memberships', label: t('account.m2mClients.scopeMemberships') },
])

const getInitialForm = (): M2MClientRequest => ({ client: '', scope: [] })
const form = ref<M2MClientRequest>(getInitialForm())

// Mutually exclusive with the specific scopes: "*" already grants them, so
// picking both is meaningless and would just be redundant to read later.
const toggleScope = (scope: string, checked: boolean) => {
  if (scope === '*') {
    form.value.scope = checked ? ['*'] : []
    return
  }

  const withoutAll = form.value.scope.filter((s) => s !== '*')
  form.value.scope = checked ? [...withoutAll, scope] : withoutAll.filter((s) => s !== scope)
}

const hasAllScope = computed(() => form.value.scope.includes('*'))

const handleModal = (isShow: boolean, client?: M2MClient) => {
  isShowModal.value = isShow
  form.value = getInitialForm()
  editingId.value = null
  flashStore.errors = null

  if (client) {
    modalType.value = 'edit'
    editingId.value = client.id
    form.value = { client: client.client, scope: [...client.scope] }
  } else {
    modalType.value = 'create'
  }
}

const onSubmit = async () => {
  if (modalType.value === 'create') {
    const created = await addNew(form.value)
    if (created) {
      handleModal(false)
      emit('created', created)
    }
    return
  }

  if (await updateById(editingId.value!, form.value)) {
    handleModal(false)
    emit('updated')
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(details: { open: boolean }) => { isShowModal = details.open }"
  >
    <DialogContent>
      <DialogTitle>
        <div class="flex items-center justify-between w-full">
          <h2 class="text-base font-medium">
            {{ modalType === 'create' ? $t('account.m2mClients.createClient') : $t('account.m2mClients.renameClient') }}
          </h2>
          <button
            type="button"
            class="text-foreground/40 hover:text-foreground/70"
            @click="handleModal(false)"
          >
            <Lucide icon="X" class="w-4 h-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="grid gap-4 mt-6">
        <div>
          <Label for="m2m-client-name">{{ $t('account.m2mClients.clientNameLabel') }} <RequiredMark /></Label>
          <Input
            id="m2m-client-name"
            v-model="form.client"
            class="mt-2"
            :placeholder="$t('account.m2mClients.clientNamePlaceholder')"
          />
          <FormFeedback feedbackKey="client" />
        </div>
        <p v-if="modalType === 'create'" class="text-xs text-foreground/50">
          {{ $t('account.m2mClients.secretHint') }}
        </p>

        <div>
          <Label>{{ $t('account.m2mClients.scopeLabel') }} <RequiredMark /></Label>
          <div class="text-xs text-foreground/40 mt-0.5">
            {{ $t('account.m2mClients.scopeHint') }}
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 mt-2">
            <label
              v-for="scope in SCOPE_OPTIONS"
              :key="scope.id"
              class="flex items-center gap-2 rounded-xl border border-foreground/10 p-3"
              :class="
                scope.id !== '*' && hasAllScope
                  ? 'opacity-40 cursor-not-allowed'
                  : 'cursor-pointer hover:border-primary/40'
              "
            >
              <input
                type="checkbox"
                class="form-checkbox shrink-0 h-4 w-4 rounded border-slate-300 text-primary"
                :checked="form.scope.includes(scope.id)"
                :disabled="scope.id !== '*' && hasAllScope"
                @change="toggleScope(scope.id, ($event.target as HTMLInputElement).checked)"
              />
              <span class="text-sm select-none">{{ scope.label }}</span>
            </label>
          </div>
          <FormFeedback feedbackKey="scope" />
        </div>
      </div>

      <div class="flex gap-2 justify-end mt-7">
        <Button look="outline" variant="ghost" @click="handleModal(false)">
          <Lucide icon="X" />
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="submitting" @click="onSubmit">
          <Lucide icon="Save" />
          {{ submitting ? $t('common.saving') : $t('common.confirm') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
