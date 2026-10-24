<script setup lang="ts">
import { NodeViewWrapper, nodeViewProps } from '@tiptap/vue-3'
import { useAttachmentUrl } from '~/composables/useAttachmentUrl'

const props = defineProps(nodeViewProps)

// The document stores the bare path; the token that makes it fetchable is
// added here, at render time. Doing it in the node view rather than by walking
// the editor's DOM afterwards means a re-render cannot drop it.
const { token, resolve } = useAttachmentUrl()
const src = computed(() => {
  void token.value // re-resolve once the token arrives
  return resolve(String(props.node.attrs['src'] ?? ''))
})

const width = computed(() => props.node.attrs['width'] as number | null)

// Shown only to someone who could actually act on it. The image looks exactly
// as interactive in read-only mode as it does in the editor, which is what
// makes "why can't I drag this" the obvious conclusion — so it says where the
// control lives instead of leaving them to guess.
const readerCouldEdit = computed(() => {
  const canEdit = props.extension.options['canEdit']
  return !props.editor.isEditable && typeof canEdit === 'function' && canEdit() === true
})

// Aspect ratio is never stored: only the width is, and the height stays auto.
// Storing both lets them drift out of proportion the moment a browser rounds
// differently, and a stretched image is worse than one that is the wrong size.
const style = computed(() => (width.value ? { width: `${width.value}px` } : {}))

const MIN_WIDTH = 80

const image = ref<HTMLImageElement | null>(null)
const resizing = ref(false)

/**
 * ProseMirror marks the image node's wrapper `draggable="true"`, so pressing
 * anywhere inside it — including on the resize handle — starts a native HTML5
 * drag of the whole node. That drag swallows the pointer sequence, and the
 * handle simply never moves.
 *
 * Suppressing it on pointerdown would be a race with the browser's own drag
 * threshold, so the wrapper stops being draggable as soon as the pointer is
 * over the handle, and becomes draggable again on the way out. Dragging the
 * image itself, from anywhere else on it, keeps working.
 */
const setWrapperDraggable = (event: PointerEvent, draggable: boolean) => {
  const wrapper = (event.currentTarget as HTMLElement).closest('.kms-image')
  if (wrapper instanceof HTMLElement) wrapper.draggable = draggable
}

/**
 * Pointer events, not mouse: the same handler then works for touch and pen
 * without a second code path, and setPointerCapture keeps the drag alive when
 * the cursor leaves the small handle.
 */
const startResize = (event: PointerEvent) => {
  if (!props.editor.isEditable || !image.value) return

  // Both matter, for different reasons: preventDefault stops the text-caret
  // drag the editor would otherwise start, stopPropagation keeps ProseMirror's
  // own mousedown handling from treating this as a click into the document.
  event.preventDefault()
  event.stopPropagation()

  const startX = event.clientX
  const startWidth = image.value.getBoundingClientRect().width
  const handle = event.currentTarget as HTMLElement
  // Resolved now, not in onUp: an event's currentTarget is nulled once its
  // dispatch finishes, so reading it later throws and leaves the wrapper
  // permanently undraggable.
  const wrapper = handle.closest('.kms-image')

  // Capture keeps the pointer bound to the handle, but a browser that refuses
  // it must not take the resize down with it — the window listeners below work
  // either way, which is why they are on the window and not on the handle.
  try {
    handle.setPointerCapture(event.pointerId)
  } catch {
    // Ignored on purpose: capture is an optimisation here, not the mechanism.
  }

  resizing.value = true

  const onMove = (move: PointerEvent) => {
    // Capped at the editor's own width so an image can never be dragged wider
    // than the column it lives in, where the overflow would be invisible until
    // someone read the document on a narrower screen.
    const max = (props.editor.view.dom as HTMLElement).clientWidth
    const next = Math.round(Math.min(Math.max(startWidth + (move.clientX - startX), MIN_WIDTH), max))
    props.updateAttributes({ width: next })
  }

  const onUp = () => {
    resizing.value = false
    if (wrapper instanceof HTMLElement) wrapper.draggable = true
    try {
      handle.releasePointerCapture(event.pointerId)
    } catch {
      // Nothing was captured; nothing to release.
    }
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
  }

  // On the window rather than the handle: a 14px target is left behind within
  // the first few pixels of any real drag, and without capture — or if capture
  // was refused — the handle would stop hearing about the pointer right there.
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

/** Back to the image's natural size — the only way out of a bad drag. */
const reset = () => props.updateAttributes({ width: null })

// Keyboard resizing, so this is not a mouse-only feature. Steps are coarse
// because a pixel at a time would take a thousand presses.
const STEP = 40
const nudge = (direction: -1 | 1) => {
  if (!image.value) return
  const current = width.value ?? image.value.getBoundingClientRect().width
  const max = (props.editor.view.dom as HTMLElement).clientWidth
  props.updateAttributes({
    width: Math.round(Math.min(Math.max(current + direction * STEP, MIN_WIDTH), max)),
  })
}
</script>

<template>
  <NodeViewWrapper
    as="div"
    class="kms-image"
    :class="[selected ? 'kms-image--selected' : '', resizing ? 'kms-image--resizing' : '']"
  >
    <img
      ref="image"
      :src="src"
      :alt="String(node.attrs.alt ?? '')"
      :title="String(node.attrs.title ?? '')"
      :style="style"
      draggable="false"
    />

    <template v-if="editor.isEditable">
      <!-- A button, not a bare div: it takes focus, so the arrow keys below
           give the same control without a pointer. -->
      <button
        type="button"
        class="kms-image__handle"
        draggable="false"
        :aria-label="$t('documents.editor.resizeImage')"
        :title="$t('documents.editor.resizeImage')"
        @pointerenter="setWrapperDraggable($event, false)"
        @pointerleave="resizing || setWrapperDraggable($event, true)"
        @dragstart.prevent.stop
        @mousedown.stop.prevent
        @pointerdown="startResize"
        @keydown.left.prevent="nudge(-1)"
        @keydown.right.prevent="nudge(1)"
        @keydown.enter.prevent="reset"
      />

      <button
        v-if="width"
        type="button"
        class="kms-image__reset kms-image__badge"
        :title="$t('documents.editor.resetImageSize')"
        @click="reset"
      >
        {{ $t('documents.editor.resetImageSize') }}
      </button>
    </template>

    <span v-else-if="readerCouldEdit" class="kms-image__badge kms-image__hint">
      {{ $t('documents.editor.resizeHint') }}
    </span>
  </NodeViewWrapper>
</template>

<style>
/* Global, not scoped: these style nodes ProseMirror creates, which scoped
   styles cannot reach. */
.kms-image {
  position: relative;
  /* fit-content, not display:table: a table shrink-wraps to the image, so the
     img's own max-width:100% would resolve against its natural width and a
     1254px screenshot would still overflow the column. This caps the wrapper
     at the column first, and the image follows it. */
  display: block;
  width: fit-content;
  max-width: 100%;
  margin: 0.75rem 0;
  line-height: 0;
}
.kms-image img {
  max-width: 100%;
  height: auto;
  border-radius: 0.5rem;
  display: block;
}
.kms-image--selected img,
.kms-image--resizing img {
  outline: 2px solid rgb(from currentColor r g b / 45%);
  outline-offset: 2px;
}
.kms-image__handle {
  position: absolute;
  right: -5px;
  bottom: -5px;
  width: 14px;
  height: 14px;
  border-radius: 3px;
  border: 2px solid var(--color-background, #fff);
  background: currentColor;
  opacity: 0;
  cursor: nwse-resize;
  touch-action: none;
  transition: opacity 0.12s;
}
.kms-image:hover .kms-image__handle,
.kms-image__handle:focus-visible,
.kms-image--selected .kms-image__handle,
.kms-image--resizing .kms-image__handle {
  opacity: 1;
}
.kms-image__badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 10px;
  line-height: 1.6;
  background: rgb(0 0 0 / 55%);
  color: #fff;
  opacity: 0;
  transition: opacity 0.12s;
  pointer-events: none;
}
.kms-image__reset {
  pointer-events: auto;
}
.kms-image:hover .kms-image__badge {
  opacity: 1;
}
</style>
