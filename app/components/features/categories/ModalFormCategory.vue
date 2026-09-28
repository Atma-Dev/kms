<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import type Category from '~/types/entities/category'
import type { CategoryRequest } from '~/services/CategoryService'

const props = defineProps<{ submitting: boolean }>()

const emit = defineEmits<{
  (e: 'create', request: CategoryRequest): void
  (e: 'update', id: string, request: CategoryRequest): void
}>()

const flashStore = useFlashStore()

const isShowModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)
// Shown read-only so it is obvious where the category is about to land —
// moving one is done from the tree, not from this form.
const parentName = ref<string | null>(null)

const getInitialForm = (): CategoryRequest => ({ name: '', parent_id: null })
const form = ref<CategoryRequest>(getInitialForm())

/**
 * `parent` is the category being added *under*; `category` is the one being
 * renamed. Passing both is how "rename a child" arrives — the parent is kept
 * so the update does not silently promote it to a root.
 */
const handleModal = (isShow: boolean, options?: { parent?: Category | null; category?: Category }) => {
  isShowModal.value = isShow
  form.value = getInitialForm()
  editingId.value = null
  parentName.value = options?.parent?.name ?? null
  flashStore.errors = null

  if (options?.category) {
    modalType.value = 'edit'
    editingId.value = options.category.id
    form.value = { name: options.category.name, parent_id: options.category.parent_id }
  } else {
    modalType.value = 'create'
    form.value.parent_id = options?.parent?.id ?? null
  }
}

const close = () => handleModal(false)

const onSubmit = () => {
  if (modalType.value === 'create') return emit('create', { ...form.value })
  emit('update', editingId.value!, { ...form.value })
}

defineExpose({ handleModal, close })
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
            {{ modalType === 'create' ? $t('categories.createCategory') : $t('categories.editCategory') }}
          </h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="close">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="mt-6 grid gap-4">
        <div v-if="parentName" class="rounded-xl bg-foreground/5 px-3 py-2 text-xs">
          {{ $t('categories.underParent', { parent: parentName }) }}
        </div>

        <div>
          <Label for="category-name">{{ $t('categories.nameLabel') }} <RequiredMark /></Label>
          <Input id="category-name" v-model="form.name" class="mt-2" :placeholder="$t('categories.namePlaceholder')" />
          <FormFeedback feedbackKey="name" />
        </div>
      </div>

      <div class="mt-7 flex justify-end gap-2">
        <Button look="outline" variant="ghost" @click="close">
          <Lucide icon="X" />
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="props.submitting" @click="onSubmit">
          <Lucide icon="Save" />
          {{ props.submitting ? $t('common.saving') : $t('common.confirm') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
