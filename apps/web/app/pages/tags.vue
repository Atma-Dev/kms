<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { useTagService } from '~/services/TagService'
import type Tag from '~/types/entities/tag'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'tags.title',
  pageSubTitle: 'tags.subtitle',
  breadcrumb: ['Tags'],
})

const authStore = useAuthStore()
const canStore = computed(() => authStore.hasPermission(Permission.MASTERS_TAGS_STORE))
const canUpdate = computed(() => authStore.hasPermission(Permission.MASTERS_TAGS_UPDATE))
const canDestroy = computed(() => authStore.hasPermission(Permission.MASTERS_TAGS_DESTROY))

const { tags, loading, submitting, getAll, addNew, renameById, deleteById } = useTagService()
const search = ref('')
const newTag = ref('')
const confirming = ref<Tag | null>(null)

// Renamed in place rather than in a modal: the only field is the name, and a
// dialog for one input is more ceremony than the edit deserves.
const renaming = ref<Tag | null>(null)
const draftName = ref('')
const renameInput = ref<HTMLInputElement | null>(null)

const startRename = async (tag: Tag) => {
  renaming.value = tag
  draftName.value = tag.name
  await nextTick()
  renameInput.value?.focus()
}

const cancelRename = () => {
  renaming.value = null
  draftName.value = ''
}

const commitRename = async () => {
  const tag = renaming.value
  const name = draftName.value.trim()
  if (!tag) return
  // Unchanged or emptied: nothing to send, and an empty name would only come
  // back as a validation error the user already knows about.
  if (!name || name === tag.name) return cancelRename()

  if (await renameById(tag.id, name)) {
    cancelRename()
    await getAll()
  }
}

// Client-side: the whole vocabulary is one small list, and the API already
// returns it in one request.
const visible = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return tags.value
  return tags.value.filter((t) => t.name.toLowerCase().includes(query))
})

onMounted(() => getAll())

const create = async () => {
  const name = newTag.value.trim()
  if (!name) return
  if (await addNew(name)) {
    newTag.value = ''
    await getAll()
  }
}

const remove = async (tag: Tag) => {
  if (await deleteById(tag.id)) {
    confirming.value = null
    await getAll()
  }
}
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 lg:col-span-8">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <div class="relative">
          <Input v-model="search" type="text" :placeholder="$t('tags.searchPlaceholder')" class="w-56 pr-8" />
          <Lucide class="absolute inset-y-0 right-0 my-auto mr-2.5 size-4 text-foreground/40" icon="Search" />
        </div>

        <div v-if="canStore" class="ml-auto flex gap-2">
          <Input
            v-model="newTag"
            type="text"
            :placeholder="$t('tags.newPlaceholder')"
            class="w-48"
            @keyup.enter="create"
          />
          <Button variant="primary" :disabled="submitting || !newTag.trim()" @click="create">
            <Lucide icon="Plus" class="mr-1.5 size-4" />
            {{ $t('tags.add') }}
          </Button>
        </div>
      </div>

      <p v-if="loading" class="py-8 text-center text-sm text-foreground/40">{{ $t('common.loading') }}</p>
      <p v-else-if="!visible.length" class="py-8 text-center text-sm text-foreground/40">{{ $t('tags.empty') }}</p>

      <ul v-else class="divide-y divide-foreground/5">
        <li v-for="tag in visible" :key="tag.id" class="flex items-center gap-3 py-2.5">
          <Lucide icon="Tag" class="size-4 shrink-0 text-foreground/30" />
          <div class="min-w-0 flex-1">
            <Input
              v-if="renaming?.id === tag.id"
              ref="renameInput"
              v-model="draftName"
              class="h-8 text-sm"
              :disabled="submitting"
              @keyup.enter="commitRename"
              @keyup.escape="cancelRename"
              @blur="commitRename"
            />
            <template v-else>
              <div class="truncate text-sm font-medium">{{ tag.name }}</div>
              <div class="truncate text-xs text-foreground/35">{{ tag.slug }}</div>
            </template>
          </div>
          <Badge variant="secondary" look="outline" class="shrink-0 text-[10px]">
            {{ $t('tags.documentCount', { count: tag.document_count ?? 0 }) }}
          </Badge>
          <button
            v-if="canUpdate && renaming?.id !== tag.id"
            type="button"
            class="shrink-0 rounded p-1.5 text-foreground/40 hover:bg-foreground/5"
            :aria-label="$t('tags.rename')"
            :title="$t('tags.rename')"
            @click="startRename(tag)"
          >
            <Lucide icon="Pencil" class="size-4" />
          </button>
          <button
            v-if="canDestroy"
            type="button"
            class="shrink-0 rounded p-1.5 text-foreground/40 hover:bg-danger/10 hover:text-danger"
            :aria-label="$t('common.delete')"
            :title="$t('common.delete')"
            @click="confirming = tag"
          >
            <Lucide icon="Trash2" class="size-4" />
          </button>
        </li>
      </ul>
    </Box>

    <Box class="col-span-12 lg:col-span-4">
      <p class="text-sm font-semibold">{{ $t('tags.aboutTitle') }}</p>
      <p class="mt-2 text-sm text-foreground/60">{{ $t('tags.aboutBody') }}</p>
    </Box>

    <!-- The count is in the message because deleting a tag unlabels every
         document that carried it, which the word "delete" does not suggest. -->
    <DialogRoot
      :open="confirming !== null"
      :onOpenChange="(d: { open: boolean }) => { if (!d.open) confirming = null }"
    >
      <DialogContent class="sm:max-w-md">
        <DialogTitle>{{ $t('tags.deleteTitle') }}</DialogTitle>
        <div class="flex flex-col items-center gap-4 py-4 text-center">
          <div class="flex h-14 w-14 items-center justify-center rounded-full bg-danger/10">
            <Lucide icon="Trash2" class="h-7 w-7 text-danger" />
          </div>
          <p class="text-sm text-foreground/70">
            {{ $t('tags.deleteConfirm', { tag: confirming?.name, count: confirming?.document_count ?? 0 }) }}
          </p>
        </div>
        <div class="mt-2 flex justify-end gap-2">
          <Button look="outline" variant="ghost" @click="confirming = null">{{ $t('common.cancel') }}</Button>
          <Button variant="danger" :disabled="submitting" @click="confirming && remove(confirming)">
            <Lucide icon="Trash2" class="mr-1.5 h-4 w-4" />
            {{ $t('common.delete') }}
          </Button>
        </div>
      </DialogContent>
    </DialogRoot>
  </div>
</template>
