<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'
import { Input } from '~/base/ui/input'
import type Tag from '~/types/entities/tag'
import { useLookupService } from '~/services/LookupService'

const props = defineProps<{
  tags: Tag[]
  editable: boolean
  submitting: boolean
}>()

const emit = defineEmits<{ (e: 'save', names: string[]): void }>()

const adding = ref(false)
const draft = ref('')
const input = ref<HTMLInputElement | null>(null)

const names = computed(() => props.tags.map((t) => t.name))

// The vocabulary is open — a new name still becomes a tag — but most of the
// time the tag already exists, and typing it again by hand is how "Deployment"
// and "deployments" end up as two entries meaning one thing.
//
// A native <datalist> is the whole feature: type-ahead, keyboard selection and
// free text in one element, with no combobox component and no dependency. The
// existing TomSelect wrapper is single-select with no create mode, so it
// cannot do this without being rewritten for every page that uses it.
const { options: knownTags, can: canSeeTags, load: loadTags } = useLookupService('tags')
const listId = useId()
const requestedTags = ref(false)

// Fetched when the field opens, not on every document view: most visits never
// add a tag, and this is one request for a list nobody asked to see.
const loadSuggestionsOnce = () => {
  if (!canSeeTags || requestedTags.value) return
  requestedTags.value = true
  void loadTags()
}

// A tag already on the document is not worth suggesting — picking it would be
// a no-op, and it crowds out the ones that are not.
const suggestions = computed(() => {
  const taken = new Set(names.value.map((n) => n.toLowerCase()))
  return knownTags.value.filter((option) => !taken.has(option.name.toLowerCase()))
})

// The API takes the whole set, so add and remove are both "here is the new
// list" — the client never has to describe a change.
const commit = (next: string[]) => emit('save', next)

const startAdding = async () => {
  adding.value = true
  loadSuggestionsOnce()
  await nextTick()
  input.value?.focus()
}

const add = () => {
  const value = draft.value.trim()
  draft.value = ''
  if (!value) return (adding.value = false)
  // A name already on the document is a no-op rather than a duplicate — the
  // API would fold it anyway, and a pointless request is a pointless toast.
  if (names.value.some((n) => n.toLowerCase() === value.toLowerCase())) {
    adding.value = false
    return
  }
  commit([...names.value, value])
  adding.value = false
}

const remove = (name: string) => commit(names.value.filter((n) => n !== name))
</script>

<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <Lucide icon="Tags" class="size-4 shrink-0 text-foreground/30" />

    <span
      v-for="tag in tags"
      :key="tag.id"
      class="inline-flex items-center gap-1 rounded-full bg-foreground/5 py-1 pl-2.5 text-xs"
      :class="editable ? 'pr-1' : 'pr-2.5'"
    >
      {{ tag.name }}
      <button
        v-if="editable"
        type="button"
        class="rounded-full p-0.5 text-foreground/40 hover:bg-danger/10 hover:text-danger"
        :disabled="submitting"
        :aria-label="$t('documents.removeTag', { tag: tag.name })"
        :title="$t('documents.removeTag', { tag: tag.name })"
        @click="remove(tag.name)"
      >
        <Lucide icon="X" class="size-3" />
      </button>
    </span>

    <span v-if="editable && adding" class="inline-flex">
      <Input
        ref="input"
        v-model="draft"
        class="h-7 w-44 text-xs"
        :list="listId"
        autocomplete="off"
        :placeholder="suggestions.length ? $t('documents.tagPlaceholderWithHints') : $t('documents.tagPlaceholder')"
        @keyup.enter="add"
        @keyup.escape="adding = false; draft = ''"
        @blur="add"
      />
      <datalist :id="listId">
        <option v-for="option in suggestions" :key="option.id" :value="option.name" />
      </datalist>
    </span>
    <button
      v-else-if="editable"
      type="button"
      class="inline-flex items-center gap-1 rounded-full border border-dashed border-foreground/20 px-2.5 py-1 text-xs text-foreground/50 hover:border-primary/40 hover:text-primary"
      :disabled="submitting"
      @click="startAdding"
    >
      <Lucide icon="Plus" class="size-3" />
      {{ $t('documents.addTag') }}
    </button>

    <span v-if="!editable && !tags.length" class="text-xs text-foreground/30">
      {{ $t('documents.noTags') }}
    </span>
  </div>
</template>
