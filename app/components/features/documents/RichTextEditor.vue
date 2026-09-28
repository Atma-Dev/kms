<script setup lang="ts">
import { EditorContent, useEditor, VueNodeViewRenderer } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import { useAttachmentService } from '~/services/AttachmentService'
import ResizableImage from '~/components/features/documents/ResizableImage.vue'
import { Lucide, type Icon } from '~/base/ui/lucide'
import type { DocumentContent } from '~/types/entities/document'

const props = defineProps<{
  modelValue: DocumentContent
  /** Read-only renders the same tree without a toolbar — one component, so a
   *  document never looks different depending on who is looking at it. */
  readonly?: boolean
  /** Enables image upload (FR-8). Images are filed against the space, so
   *  without one there is nowhere to put them and the affordance is hidden. */
  spaceId?: string
  /** Whether this reader could edit the document if they chose to. Read-only
   *  images use it to say how to resize them — telling someone to click a
   *  button they are not allowed to press would be worse than saying nothing. */
  canEdit?: boolean
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: DocumentContent): void }>()

// Image with a width someone can drag, rendered by a Vue component so the
// handle and the token-bearing src live with the node instead of being applied
// to the DOM afterwards.
//
// Only the width is stored. Height stays auto, so an image cannot be dragged
// out of proportion and a browser rounding differently cannot stretch it.
const ResizableImageExtension = Image.extend({
  // A getter, not a value: it is read when a node renders, so it reflects the
  // prop as it stands rather than as it was when the editor was built.
  addOptions() {
    return { ...this.parent?.(), canEdit: () => props.canEdit === true }
  },
  addAttributes() {
    return {
      ...this.parent?.(),
      width: {
        default: null,
        // Rendered as an attribute rather than an inline style: the document
        // is not HTML, and a style string would be markup smuggled into what
        // is meant to be a portable node tree.
        parseHTML: (element: HTMLElement) => {
          const value = Number(element.getAttribute('width'))
          return Number.isFinite(value) && value > 0 ? value : null
        },
        renderHTML: (attributes: Record<string, unknown>) =>
          attributes['width'] ? { width: String(attributes['width']) } : {},
      },
    }
  },
  addNodeView() {
    return VueNodeViewRenderer(ResizableImage)
  },
}).configure({ inline: false, canEdit: () => props.canEdit === true })

const editor = useEditor({
  content: props.modelValue,
  editable: !props.readonly,
  extensions: [StarterKit, ResizableImageExtension],
  editorProps: {
    attributes: {
      // `prose` styling is applied on the wrapper below rather than here, so
      // the viewer and the editor share one set of rules.
      class: 'focus:outline-none min-h-[300px]',
    },
  },
  onUpdate: ({ editor }) => emit('update:modelValue', editor.getJSON() as DocumentContent),
})

// The parent replaces content wholesale after a save or a rollback (the API
// answers every write with the whole document). Writing it back into the
// editor would move the caret on every keystroke, so this only runs when the
// incoming tree differs from what the editor already holds.
watch(
  () => props.modelValue,
  (value) => {
    if (!editor.value) return
    if (JSON.stringify(editor.value.getJSON()) === JSON.stringify(value)) return
    editor.value.commands.setContent(value as never, { emitUpdate: false })
  },
)

watch(
  () => props.readonly,
  (readonly) => editor.value?.setEditable(!readonly),
)

// --- Inline images (FR-8) ---
//
// What is stored in the content is the bare `/api/attachments/{id}/download`
// path. The token that makes it fetchable is added by ResizableImage at render
// time and never written into the document: a token in stored content would
// leak with every copy of it and expire minutes later, leaving a permanently
// broken image.
const { uploadInlineImage } = useAttachmentService()
const imageInput = ref<HTMLInputElement | null>(null)
const uploadingImage = ref(false)

const insertImage = async (file: File) => {
  if (!props.spaceId) return
  uploadingImage.value = true
  try {
    const stored = await uploadInlineImage(props.spaceId, file)
    if (!stored) return
    // The stored path goes into the document; the node view is what turns it
    // into a fetchable src when it renders.
    editor.value?.chain().focus().setImage({ src: stored }).run()
  } finally {
    uploadingImage.value = false
  }
}

const onPickImage = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) await insertImage(file)
}

/** Drag-and-drop and paste, which is how FR-8 says images arrive. */
const onDrop = (event: DragEvent) => {
  const file = [...(event.dataTransfer?.files ?? [])].find((f) => f.type.startsWith('image/'))
  if (!file || props.readonly || !props.spaceId) return
  event.preventDefault()
  void insertImage(file)
}

const onPaste = (event: ClipboardEvent) => {
  const file = [...(event.clipboardData?.files ?? [])].find((f) => f.type.startsWith('image/'))
  if (!file || props.readonly || !props.spaceId) return
  event.preventDefault()
  void insertImage(file)
}

onBeforeUnmount(() => editor.value?.destroy())

interface ToolbarItem {
  icon: Icon
  label: string
  run: () => void
  active: () => boolean
}

const { t } = useI18n()

const toolbar = computed<ToolbarItem[]>(() => {
  const chain = () => editor.value!.chain().focus()
  return [
    { icon: 'Heading1', label: t('documents.editor.heading1'), run: () => chain().toggleHeading({ level: 1 }).run(), active: () => !!editor.value?.isActive('heading', { level: 1 }) },
    { icon: 'Heading2', label: t('documents.editor.heading2'), run: () => chain().toggleHeading({ level: 2 }).run(), active: () => !!editor.value?.isActive('heading', { level: 2 }) },
    { icon: 'Bold', label: t('documents.editor.bold'), run: () => chain().toggleBold().run(), active: () => !!editor.value?.isActive('bold') },
    { icon: 'Italic', label: t('documents.editor.italic'), run: () => chain().toggleItalic().run(), active: () => !!editor.value?.isActive('italic') },
    { icon: 'Strikethrough', label: t('documents.editor.strike'), run: () => chain().toggleStrike().run(), active: () => !!editor.value?.isActive('strike') },
    { icon: 'List', label: t('documents.editor.bulletList'), run: () => chain().toggleBulletList().run(), active: () => !!editor.value?.isActive('bulletList') },
    { icon: 'ListOrdered', label: t('documents.editor.orderedList'), run: () => chain().toggleOrderedList().run(), active: () => !!editor.value?.isActive('orderedList') },
    { icon: 'Quote', label: t('documents.editor.blockquote'), run: () => chain().toggleBlockquote().run(), active: () => !!editor.value?.isActive('blockquote') },
    { icon: 'Code', label: t('documents.editor.codeBlock'), run: () => chain().toggleCodeBlock().run(), active: () => !!editor.value?.isActive('codeBlock') },
  ]
})
</script>

<template>
  <div class="rounded-xl border border-foreground/10">
    <div v-if="!readonly" class="flex flex-wrap gap-1 border-b border-foreground/10 p-2">
      <button
        v-for="item in toolbar"
        :key="item.icon"
        type="button"
        :title="item.label"
        :aria-label="item.label"
        :aria-pressed="item.active()"
        class="rounded-lg p-2 hover:bg-foreground/5"
        :class="item.active() ? 'bg-primary/10 text-primary' : 'text-foreground/60'"
        @click="item.run()"
      >
        <Lucide :icon="item.icon" class="size-4" />
      </button>

      <template v-if="spaceId">
        <input ref="imageInput" type="file" accept="image/*" class="hidden" @change="onPickImage" />
        <button
          type="button"
          :title="$t('documents.editor.image')"
          :aria-label="$t('documents.editor.image')"
          class="rounded-lg p-2 text-foreground/60 hover:bg-foreground/5 disabled:opacity-40"
          :disabled="uploadingImage"
          @click="imageInput?.click()"
        >
          <Lucide :icon="uploadingImage ? 'LoaderCircle' : 'ImagePlus'" class="size-4" :class="uploadingImage ? 'animate-spin' : ''" />
        </button>
      </template>

      <div class="ml-auto flex gap-1">
        <button
          type="button"
          :title="$t('documents.editor.undo')"
          :aria-label="$t('documents.editor.undo')"
          class="rounded-lg p-2 text-foreground/60 hover:bg-foreground/5"
          @click="editor?.chain().focus().undo().run()"
        >
          <Lucide icon="Undo2" class="size-4" />
        </button>
        <button
          type="button"
          :title="$t('documents.editor.redo')"
          :aria-label="$t('documents.editor.redo')"
          class="rounded-lg p-2 text-foreground/60 hover:bg-foreground/5"
          @click="editor?.chain().focus().redo().run()"
        >
          <Lucide icon="Redo2" class="size-4" />
        </button>
      </div>
    </div>

    <EditorContent
      :editor="editor"
      class="kms-prose p-4"
      @drop="onDrop"
      @paste="onPaste"
    />
  </div>
</template>

<style>
/* Scoped styles cannot reach TipTap's generated nodes, so these are global and
   namespaced by .kms-prose instead. */
.kms-prose h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 1rem 0 0.5rem;
}
.kms-prose h2 {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0.875rem 0 0.5rem;
}
.kms-prose h3 {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0.75rem 0 0.5rem;
}
.kms-prose p {
  margin: 0.5rem 0;
  line-height: 1.7;
}
.kms-prose ul,
.kms-prose ol {
  margin: 0.5rem 0 0.5rem 1.25rem;
}
.kms-prose ul {
  list-style: disc;
}
.kms-prose ol {
  list-style: decimal;
}
.kms-prose li {
  margin: 0.25rem 0;
}
.kms-prose blockquote {
  border-left: 3px solid rgb(from currentColor r g b / 20%);
  padding-left: 0.875rem;
  margin: 0.75rem 0;
  opacity: 0.85;
}
.kms-prose pre {
  background: rgb(from currentColor r g b / 6%);
  border-radius: 0.5rem;
  padding: 0.875rem;
  margin: 0.75rem 0;
  overflow-x: auto;
  font-size: 0.85em;
}
.kms-prose code {
  font-family: ui-monospace, SFMono-Regular, monospace;
}
.kms-prose :first-child {
  margin-top: 0;
}
</style>
