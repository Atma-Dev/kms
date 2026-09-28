<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Textarea } from '~/base/ui/textarea'
import { Lucide } from '~/base/ui/lucide'
import type Space from '~/types/entities/space'
import type { ProtectionLevel } from '~/types/entities/protection'
import { useProtectionService } from '~/services/ProtectionService'
import { type SpaceRequest, useSpaceService } from '~/services/SpaceService'

const emit = defineEmits<{ (e: 'saved'): void }>()

const { addNew, updateById, submitting } = useSpaceService()
const { setSpaceLevel } = useProtectionService()
const flashStore = useFlashStore()

const isShowModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)

// description and icon are nullable columns, but an input can only bind a
// string — they are held as '' here and sent as null, so a cleared field
// clears the column instead of storing an empty string.
interface SpaceForm {
  name: string
  description: string
  icon: string
  is_archived?: boolean
  /**
   * Modul 4 FR-6. Held here but saved through its own endpoint, because
   * PUT /spaces/{id} is a full replace — folding one enum into it would send
   * this client's copy of the name and description along with it.
   */
  protection_level: ProtectionLevel
}

const getInitialForm = (): SpaceForm => ({ name: '', description: '', icon: '', protection_level: 'normal' })
const form = ref<SpaceForm>(getInitialForm())

const toRequest = (): SpaceRequest => ({
  name: form.value.name,
  description: form.value.description || null,
  icon: form.value.icon || null,
  ...(form.value.is_archived === undefined ? {} : { is_archived: form.value.is_archived }),
})

// What the space was on when the dialog opened, so the second request only
// goes out when the level actually changed.
const savedLevel = ref<ProtectionLevel>('normal')

const { t } = useI18n()
const protectionOptions = computed(() => [
  { id: 'normal', name: t('documents.protection.normal') },
  { id: 'strict', name: t('documents.protection.strict') },
])

const handleModal = (isShow: boolean, space?: Space) => {
  isShowModal.value = isShow
  form.value = getInitialForm()
  editingId.value = null
  flashStore.errors = null

  if (space) {
    modalType.value = 'edit'
    editingId.value = space.id
    form.value = {
      name: space.name,
      description: space.description ?? '',
      icon: space.icon ?? '',
      is_archived: space.is_archived,
      protection_level: space.protection_level,
    }
    savedLevel.value = space.protection_level
  } else {
    modalType.value = 'create'
  }
}

const onSubmit = async () => {
  // The slug is derived from the name server-side, so it is never a field
  // here — two places naming the same URL is how they end up disagreeing.
  if (modalType.value === 'create') {
    // A new space takes the server's default and nothing else: the level is
    // offered on edit, where there is a space to set it on.
    if (await addNew(toRequest())) {
      handleModal(false)
      emit('saved')
    }
    return
  }

  if (!(await updateById(editingId.value!, toRequest()))) return

  if (form.value.protection_level !== savedLevel.value) {
    await setSpaceLevel(editingId.value!, form.value.protection_level)
  }

  handleModal(false)
  emit('saved')
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
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">
            {{ modalType === 'create' ? $t('spaces.createSpace') : $t('spaces.editSpace') }}
          </h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="mt-6 grid gap-4">
        <div>
          <Label for="space-name">{{ $t('spaces.nameLabel') }} <RequiredMark /></Label>
          <Input id="space-name" v-model="form.name" class="mt-2" :placeholder="$t('spaces.namePlaceholder')" />
          <FormFeedback feedbackKey="name" />
        </div>

        <div>
          <Label for="space-description">{{ $t('spaces.descriptionLabel') }}</Label>
          <Textarea
            id="space-description"
            v-model="form.description"
            class="mt-2"
            rows="3"
            :placeholder="$t('spaces.descriptionPlaceholder')"
          />
          <FormFeedback feedbackKey="description" />
        </div>

        <div>
          <Label for="space-icon">{{ $t('spaces.iconLabel') }}</Label>
          <div class="mt-2">
            <IconSelect id="space-icon" v-model="form.icon" :placeholder="$t('spaces.iconPlaceholder')" />
          </div>
          <div class="mt-1 text-xs text-foreground/40">{{ $t('spaces.iconHint') }}</div>
          <FormFeedback feedbackKey="icon" />
        </div>

        <!-- Modul 4 FR-6. Only on edit: a new space takes the server's
             default, and there is nothing to set it on until it exists. -->
        <div v-if="modalType === 'edit'">
          <Label>{{ $t('spaces.protectionLabel') }} <RequiredMark /></Label>
          <div class="mt-2">
            <TomSelect
              v-model="form.protection_level"
              :options="protectionOptions"
              :placeholder="$t('spaces.protectionPlaceholder')"
            />
          </div>
          <div class="mt-1 text-xs text-foreground/40">{{ $t('spaces.protectionHint') }}</div>
          <FormFeedback feedbackKey="protection_level" />
        </div>
      </div>

      <div class="mt-7 flex justify-end gap-2">
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
