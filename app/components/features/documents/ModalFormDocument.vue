<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import type Category from '~/types/entities/category'
import { useDocumentService } from '~/services/DocumentService'

const props = defineProps<{ spaceId: string; tree: Category[] }>()
const emit = defineEmits<{ (e: 'created', id: string): void }>()

const { addNew, submitting } = useDocumentService()
const flashStore = useFlashStore()

const isShowModal = ref(false)
const title = ref('')
// TomSelect writes a plain string, so the empty choice is '' here and
// becomes null on the way to the API — the column is nullable, not empty.
const categoryId = ref('')

// Flattened with each option carrying its full path, so two branches' "Notes"
// are distinguishable — the same shape the categories lookup returns.
const categoryOptions = computed(() => {
  const options: { id: string; name: string }[] = []
  const walk = (nodes: Category[], prefix: string) => {
    for (const node of nodes) {
      const path = prefix ? `${prefix} / ${node.name}` : node.name
      options.push({ id: node.id, name: path })
      walk(node.children, path)
    }
  }
  walk(props.tree, '')
  return options
})

const handleModal = (isShow: boolean) => {
  isShowModal.value = isShow
  title.value = ''
  categoryId.value = ''
  flashStore.errors = null
}

const onSubmit = async () => {
  // Starts empty and always as a draft — the API refuses to take a status
  // here, and the content is written on the document page itself rather than
  // in a cramped modal.
  const created = await addNew(props.spaceId, {
    title: title.value,
    category_id: categoryId.value || null,
    content: { type: 'doc', content: [{ type: 'paragraph' }] },
  })

  if (created) {
    handleModal(false)
    emit('created', created.id)
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }">
    <DialogContent>
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.createDocument') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="mt-6 grid gap-4">
        <div>
          <Label for="document-title">{{ $t('documents.titleLabel') }} <RequiredMark /></Label>
          <Input
            id="document-title"
            v-model="title"
            class="mt-2"
            :placeholder="$t('documents.titlePlaceholder')"
            @keyup.enter="onSubmit"
          />
          <FormFeedback feedbackKey="title" />
        </div>

        <div>
          <Label>{{ $t('documents.categoryLabel') }}</Label>
          <TomSelect
            v-model="categoryId"
            class="mt-2"
            :options="categoryOptions"
            :placeholder="$t('documents.noCategory')"
          />
          <FormFeedback feedbackKey="category_id" />
        </div>
      </div>

      <div class="mt-7 flex justify-end gap-2">
        <Button look="outline" variant="ghost" @click="handleModal(false)">
          <Lucide icon="X" />
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="submitting || !title" @click="onSubmit">
          <Lucide icon="Save" />
          {{ submitting ? $t('common.saving') : $t('documents.createAndEdit') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
