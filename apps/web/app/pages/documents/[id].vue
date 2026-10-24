<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useDocumentService } from '~/services/DocumentService'
import type { DocumentAction, DocumentContent } from '~/types/entities/document'
import RichTextEditor from '~/components/features/documents/RichTextEditor.vue'
import ModalTransitionReason from '~/components/features/documents/ModalTransitionReason.vue'
import ModalVersionDiff from '~/components/features/documents/ModalVersionDiff.vue'
import ModalDocumentAccess from '~/components/features/documents/ModalDocumentAccess.vue'
import ModalShareLink from '~/components/features/documents/ModalShareLink.vue'
import ModalAccessLog from '~/components/features/documents/ModalAccessLog.vue'
import ModalProtectionSetting from '~/components/features/documents/ModalProtectionSetting.vue'
import DocumentProtection from '~/components/features/documents/DocumentProtection.vue'
import DocumentPages from '~/components/features/documents/DocumentPages.vue'
import { useProtectionService } from '~/services/ProtectionService'
import DocumentTags from '~/components/features/documents/DocumentTags.vue'
import DocumentAttachments from '~/components/features/documents/DocumentAttachments.vue'
import RelatedDocuments from '~/components/features/documents/RelatedDocuments.vue'
import ModalDelete from '~/components/ModalDelete.vue'
import { Permission } from '~/enums/Permission'
import { actionNeedsReason, actionStyle, statusVariant } from '~/utils/document'

definePageMeta({
  title: 'documents.detailTitle',
  breadcrumb: ['Documents', 'Detail'],
})

const route = useRoute()
const documentId = computed(() => route.params.id as string)

const authStore = useAuthStore()
const canDestroy = computed(() => authStore.hasPermission(Permission.DOCUMENTS_DESTROY))
const canManageAccess = computed(() => authStore.hasPermission(Permission.DOCUMENTS_ACCESS_MANAGE))
const canManageProtection = computed(() => authStore.hasPermission(Permission.DOCUMENTS_PROTECTION_MANAGE))
const canShare = computed(() => authStore.hasPermission(Permission.DOCUMENTS_SHARE))
const canReadAccessLog = computed(() => authStore.hasPermission(Permission.DOCUMENTS_ACCESS_LOG_SHOW))

const {
  document,
  versions,
  loading,
  submitting,
  getById,
  updateById,
  transition,
  deleteById,
  getVersions,
  compare,
  rollback,
  syncTags,
  setPinned,
} = useDocumentService()

const { exportPdf, reportPrint, submitting: protecting } = useProtectionService()

/**
 * The client half of FR-1: the reader's own identity, overlaid on what they
 * are looking at.
 *
 * Deliberately built here from the logged-in account rather than sent by the
 * API. It is a deterrent either way — anyone with dev tools can delete the
 * overlay — so asking the server for a string it would only echo back would
 * buy nothing. What cannot be forged is the watermark on the PDF and on the
 * strict viewer's page images, and both of those are drawn server-side.
 */
const watermark = computed(() => {
  const email = authStore.user?.email
  if (!email) return ''
  return `${email} · ${new Date().toLocaleString()}`
})

// Only ever true for a reader: an editor keeps the text, since withholding it
// from someone who can rewrite the document protects nothing.
const showPages = computed(() => document.value?.content_hidden === true)

const onExportPdf = () => {
  if (document.value) exportPdf(documentId.value, document.value.slug)
}

/**
 * FR-4's one client-reported access. A browser print never reaches the API, so
 * without this the trail would show every export and no prints at all — and
 * printing is the easier way to walk out with a document.
 *
 * It is a hint rather than evidence: a reader who blocks the request simply
 * does not appear. Everything else in the log is observed server-side.
 */
onMounted(() => {
  if (!import.meta.client) return
  const onBeforePrint = () => reportPrint(documentId.value)
  window.addEventListener('beforeprint', onBeforePrint)
  onBeforeUnmount(() => window.removeEventListener('beforeprint', onBeforePrint))
})

const editing = ref(false)
const form = reactive({ title: '', content: {} as DocumentContent, changeNote: '' })

const load = async () => {
  await getById(documentId.value)
  await getVersions(documentId.value)
}
onMounted(load)

const startEditing = () => {
  if (!document.value) return
  form.title = document.value.title
  form.content = document.value.content
  form.changeNote = ''
  editing.value = true
}

const cancelEditing = () => {
  editing.value = false
}

const save = async () => {
  if (!document.value) return
  const saved = await updateById(documentId.value, {
    title: form.title,
    category_id: document.value.category_id,
    content: form.content,
    // Only meaningful once the document has been published — before that the
    // API takes no snapshot, so there is nothing for the note to describe.
    change_note: form.changeNote || null,
  })
  if (saved) {
    editing.value = false
    await getVersions(documentId.value)
  }
}

const onSaveTags = (names: string[]) => syncTags(documentId.value, names)

const togglePin = () => setPinned(documentId.value, !document.value?.is_pinned)

const reasonModal = ref<InstanceType<typeof ModalTransitionReason> | null>(null)
const accessModal = ref<InstanceType<typeof ModalDocumentAccess> | null>(null)
const diffModal = ref<InstanceType<typeof ModalVersionDiff> | null>(null)
const deleteModal = ref<InstanceType<typeof ModalDelete> | null>(null)
const shareModal = ref<InstanceType<typeof ModalShareLink> | null>(null)
const accessLogModal = ref<InstanceType<typeof ModalAccessLog> | null>(null)
const protectionModal = ref<InstanceType<typeof ModalProtectionSetting> | null>(null)

const runAction = async (action: DocumentAction) => {
  if (actionNeedsReason(action)) return reasonModal.value?.handleModal(true, action)
  if (await transition(documentId.value, action)) await getVersions(documentId.value)
}

const onReasonConfirmed = async (action: DocumentAction, reason: string) => {
  if (await transition(documentId.value, action, reason)) {
    reasonModal.value?.handleModal(false)
    await getVersions(documentId.value)
  }
}

const onCompare = async (from: string, to: string) => {
  diffModal.value?.setResult(await compare(documentId.value, from, to))
}

const onRollback = async (versionNumber: number) => {
  if (await rollback(documentId.value, versionNumber)) await getVersions(documentId.value)
}

const onDelete = async () => {
  if (await deleteById(documentId.value)) {
    await navigateTo(document.value ? `/spaces/${document.value.space_id}` : '/spaces')
  }
}
</script>

<template>
  <div v-if="loading && !document" class="py-16 text-center text-sm text-foreground/40">
    {{ $t('common.loading') }}
  </div>

  <div v-else-if="document" class="grid grid-cols-12 gap-6">
    <div class="col-span-12">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="flex min-w-0 items-center gap-3">
          <NuxtLink
            :to="`/spaces/${document.space_id}`"
            class="rounded-lg p-2 text-foreground/50 hover:bg-foreground/5"
          >
            <Lucide icon="ArrowLeft" class="size-4" />
          </NuxtLink>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-lg font-medium">{{ document.title }}</h2>
              <Badge :variant="statusVariant(document.status)" look="outline" class="text-[10px]">
                {{ $t(`documents.status.${document.status}`) }}
              </Badge>
              <Badge v-if="document.is_pinned" variant="primary" look="outline" class="text-[10px]">
                {{ $t('documents.pinned') }}
              </Badge>
              <!-- Says only that a lock is on. Who is on the list is gated
                   separately, so this never leaks the allowlist. -->
              <Badge v-if="document.is_restricted" variant="warning" look="outline" class="text-[10px]">
                <Lucide icon="Lock" class="mr-1 size-3" />
                {{ $t('documents.restricted') }}
              </Badge>
              <!-- Only strict earns a badge. Normal is the default every
                   document is at, and a badge on every one of them says
                   nothing. -->
              <Badge
                v-if="document.protection_level === 'strict'"
                variant="danger"
                look="outline"
                class="text-[10px]"
              >
                <Lucide icon="ShieldCheck" class="mr-1 size-3" />
                {{ $t('documents.protection.strict') }}
              </Badge>
            </div>
            <p class="mt-0.5 text-xs text-foreground/40">
              /{{ document.slug }} · {{ $t('documents.updatedAt', { at: document.updated_at }) }} ·
              {{ $t('documents.viewCount', { count: document.view_count }) }}
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <!-- Rendered straight from the API's available_actions: it already
               filtered them by status and by what this caller holds, so
               re-deriving the rules here would only let the two drift. -->
          <Button
            v-for="action in document.available_actions"
            :key="action"
            size="sm"
            :variant="actionStyle[action].variant"
            :disabled="submitting || editing"
            @click="runAction(action)"
          >
            <Lucide :icon="actionStyle[action].icon" class="mr-1.5 size-4" />
            {{ $t(`documents.action.${action}`) }}
          </Button>

          <Button
            v-if="document.can_edit && !editing"
            size="sm"
            look="outline"
            :variant="document.is_pinned ? 'primary' : 'ghost'"
            :disabled="submitting"
            :title="document.is_pinned ? $t('documents.unpin') : $t('documents.pin')"
            @click="togglePin"
          >
            <Lucide :icon="document.is_pinned ? 'PinOff' : 'Pin'" class="size-4" />
          </Button>

          <Button
            v-if="document.can_edit && !editing"
            size="sm"
            look="outline"
            variant="ghost"
            @click="startEditing"
          >
            <Lucide icon="Pencil" class="mr-1.5 size-4" />
            {{ $t('common.edit') }}
          </Button>

          <Button
            v-if="canManageAccess && !editing"
            size="sm"
            look="outline"
            variant="ghost"
            @click="accessModal?.handleModal(true)"
          >
            <Lucide :icon="document.is_restricted ? 'Lock' : 'LockOpen'" class="mr-1.5 size-4" />
            {{ $t('documents.access.button') }}
          </Button>

          <Button
            v-if="!editing"
            size="sm"
            look="outline"
            variant="ghost"
            :disabled="protecting"
            @click="onExportPdf"
          >
            <Lucide icon="FileDown" class="mr-1.5 size-4" />
            {{ $t('documents.protection.exportPdf') }}
          </Button>

          <Button
            v-if="canShare && !editing && document.status === 'published'"
            size="sm"
            look="outline"
            variant="ghost"
            @click="shareModal?.handleModal(true)"
          >
            <Lucide icon="Link" class="mr-1.5 size-4" />
            {{ $t('documents.share.button') }}
          </Button>

          <Button
            v-if="canManageProtection && !editing"
            size="sm"
            look="outline"
            variant="ghost"
            @click="protectionModal?.handleModal(true)"
          >
            <Lucide icon="ShieldCheck" class="mr-1.5 size-4" />
            {{ $t('documents.protection.button') }}
          </Button>

          <Button
            v-if="canReadAccessLog && !editing"
            size="sm"
            look="outline"
            variant="ghost"
            @click="accessLogModal?.handleModal(true)"
          >
            <Lucide icon="History" class="mr-1.5 size-4" />
            {{ $t('documents.accessLog.button') }}
          </Button>

          <Button
            v-if="canDestroy && !editing"
            size="sm"
            look="outline"
            variant="ghost"
            class="text-danger"
            @click="deleteModal?.handleModal(true)"
          >
            <Lucide icon="Trash2" class="size-4" />
          </Button>
        </div>
      </div>
    </div>

    <div class="col-span-12 -mt-2">
      <DocumentTags
        :tags="document.tags"
        :editable="document.can_edit"
        :submitting="submitting"
        @save="onSaveTags"
      />
    </div>

    <Box class="col-span-12 lg:col-span-9">
      <template v-if="editing">
        <Input v-model="form.title" class="mb-3 text-lg font-medium" :placeholder="$t('documents.titlePlaceholder')" />
        <FormFeedback feedbackKey="title" />

        <RichTextEditor v-model="form.content" :space-id="document.space_id" />

        <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
          <div v-if="document.published_at" class="min-w-0 flex-1">
            <Input v-model="form.changeNote" :placeholder="$t('documents.changeNotePlaceholder')" />
            <p class="mt-1 text-xs text-foreground/40">{{ $t('documents.changeNoteHint') }}</p>
          </div>
          <div class="flex gap-2">
            <Button look="outline" variant="ghost" :disabled="submitting" @click="cancelEditing">
              {{ $t('common.cancel') }}
            </Button>
            <Button variant="primary" :disabled="submitting" @click="save">
              <Lucide icon="Save" class="mr-1.5 size-4" />
              {{ submitting ? $t('common.saving') : $t('common.save') }}
            </Button>
          </div>
        </div>
      </template>

      <!-- A strict document sends no text to a reader, so there is nothing
           here to render as prose — the pages are images with the watermark
           already in them. An editor still gets the editor: they can rewrite
           the document, so withholding its text protects nothing. -->
      <DocumentPages v-else-if="showPages" :document-id="documentId" />

      <DocumentProtection
        v-else
        :text="watermark"
        :protect="document.protection_level === 'strict'"
      >
        <RichTextEditor
          :model-value="document.content"
          :can-edit="document.can_edit"
          readonly
        />
      </DocumentProtection>
    </Box>

    <Box class="col-span-12 lg:col-span-3">
      <DocumentAttachments :document-id="documentId" :can-edit="document.can_edit" />
    </Box>

    <Box class="col-span-12 lg:col-span-3">
      <RelatedDocuments :document-id="documentId" />
    </Box>

    <Box class="col-span-12 lg:col-span-3 lg:col-start-10">
      <div class="mb-3 flex items-center justify-between">
        <p class="text-sm font-semibold">{{ $t('documents.versionsTitle') }}</p>
        <Button v-if="versions.length" size="sm" look="outline" variant="ghost" @click="diffModal?.handleModal(true)">
          <Lucide icon="GitCompare" class="mr-1.5 size-4" />
          {{ $t('documents.compare') }}
        </Button>
      </div>

      <!-- A document that has never been published has no versions, by design:
           a draft nobody has seen has no reader to protect. -->
      <p v-if="!versions.length" class="text-sm text-foreground/40">
        {{ document.published_at ? $t('documents.noVersionsYet') : $t('documents.versionsAfterPublish') }}
      </p>

      <ul v-else class="divide-y divide-foreground/5">
        <li v-for="version in versions" :key="version.id" class="py-2.5">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="text-sm font-medium">v{{ version.version_number }}</div>
              <p class="mt-0.5 line-clamp-2 text-xs text-foreground/50">{{ version.change_note || '—' }}</p>
              <p class="mt-0.5 text-[11px] text-foreground/35">{{ version.created_at }}</p>
            </div>
            <button
              v-if="document.can_edit"
              type="button"
              class="shrink-0 rounded p-1.5 text-foreground/50 hover:bg-foreground/5"
              :disabled="submitting"
              :aria-label="$t('documents.rollbackTo', { version: version.version_number })"
              :title="$t('documents.rollbackTo', { version: version.version_number })"
              @click="onRollback(version.version_number)"
            >
              <Lucide icon="RotateCcw" class="size-4" />
            </button>
          </div>
        </li>
      </ul>
    </Box>

    <ModalTransitionReason ref="reasonModal" :submitting="submitting" @confirm="onReasonConfirmed" />
    <ModalVersionDiff ref="diffModal" :versions="versions" @compare="onCompare" />
    <ModalDocumentAccess ref="accessModal" :document-id="documentId" @changed="load" />
    <ModalShareLink ref="shareModal" :document-id="documentId" />
    <ModalAccessLog ref="accessLogModal" :document-id="documentId" />
    <ModalProtectionSetting
      ref="protectionModal"
      :document-id="documentId"
      :current="document.protection_level_override"
      :effective="document.protection_level"
      @saved="load"
    />
    <ModalDelete ref="deleteModal" @submit="onDelete" />
  </div>
</template>
