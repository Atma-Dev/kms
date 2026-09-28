<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Label } from '~/base/ui/label'
import { Lucide } from '~/base/ui/lucide'
import { useProtectionService } from '~/services/ProtectionService'
import type { ViewSession } from '~/types/entities/protection'

/**
 * Expiring share links (FR-5): a document handed to somebody who has no
 * account, for as long as the link lasts and no longer.
 */
const props = defineProps<{ documentId: string }>()

const { sessions, loading, submitting, getViewSessions, createViewSession, revokeViewSession } =
  useProtectionService()

const isShowModal = ref(false)
const label = ref('')
const expiresInHours = ref(72)

/**
 * The token, held only between creating a link and closing this dialog.
 *
 * The API stores just its hash, so this is the one and only chance anyone has
 * to copy it. Closing the dialog throws it away — which is the honest
 * behaviour, since the alternative is a UI that implies it can be looked up
 * again.
 */
const freshLink = ref('')
const copied = ref(false)

const { t } = useI18n()
const expiryOptions = computed(() => [
  { id: '24', name: t('documents.share.expiry1Day') },
  { id: '72', name: t('documents.share.expiry3Days') },
  { id: '168', name: t('documents.share.expiry1Week') },
  { id: '720', name: t('documents.share.expiry30Days') },
])

const handleModal = async (isShow: boolean) => {
  isShowModal.value = isShow
  if (!isShow) {
    freshLink.value = ''
    copied.value = false
    return
  }
  label.value = ''
  expiresInHours.value = 72
  await getViewSessions(props.documentId)
}

const create = async () => {
  const created = await createViewSession(
    props.documentId,
    Number(expiresInHours.value),
    label.value.trim() || null,
  )
  if (!created) return

  freshLink.value = `${window.location.origin}/shared/${created.token}`
  copied.value = false
  label.value = ''
  await getViewSessions(props.documentId)
}

const copy = async () => {
  try {
    await navigator.clipboard.writeText(freshLink.value)
    copied.value = true
  } catch {
    // Clipboard access is refused in some browsers and on insecure origins.
    // The link is on screen and selectable, so there is nothing to recover
    // from — only a button that did not light up.
  }
}

const revoke = async (session: ViewSession) => {
  if (await revokeViewSession(props.documentId, session.id)) {
    await getViewSessions(props.documentId)
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => handleModal(d.open)">
    <DialogContent class="sm:max-w-lg">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.share.title') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <p class="mt-3 rounded-xl bg-foreground/5 px-3 py-2 text-xs text-foreground/60">
        {{ $t('documents.share.hint') }}
      </p>

      <!-- Shown once. Says so, because the API keeps only the hash and nothing
           can recover this afterwards. -->
      <div v-if="freshLink" class="mt-4 rounded-xl border border-primary/30 bg-primary/5 p-3">
        <p class="text-xs font-medium text-primary">{{ $t('documents.share.onceOnly') }}</p>
        <div class="mt-2 flex items-center gap-2">
          <Input :model-value="freshLink" readonly class="text-xs" @focus="($event.target as HTMLInputElement).select()" />
          <Button size="sm" variant="primary" @click="copy">
            <Lucide :icon="copied ? 'Check' : 'Copy'" class="size-4" />
          </Button>
        </div>
      </div>

      <div class="mt-4 grid gap-3">
        <div>
          <Label for="share-label">{{ $t('documents.share.labelLabel') }}</Label>
          <Input
            id="share-label"
            v-model="label"
            class="mt-2"
            :placeholder="$t('documents.share.labelPlaceholder')"
          />
          <FormFeedback feedbackKey="label" />
        </div>

        <div>
          <Label>{{ $t('documents.share.expiryLabel') }} <RequiredMark /></Label>
          <div class="mt-2">
            <TomSelect v-model="expiresInHours" :options="expiryOptions" :placeholder="$t('documents.share.expiryPlaceholder')" />
          </div>
          <FormFeedback feedbackKey="expires_in_hours" />
        </div>

        <Button variant="primary" :disabled="submitting" @click="create">
          <Lucide icon="Link" class="mr-1.5 size-4" />
          {{ $t('documents.share.create') }}
        </Button>
      </div>

      <div class="mt-5">
        <p class="mb-2 text-sm font-semibold">{{ $t('documents.share.liveTitle') }}</p>
        <p v-if="loading" class="text-sm text-foreground/40">{{ $t('common.loading') }}</p>
        <p v-else-if="!sessions.length" class="text-sm text-foreground/40">{{ $t('documents.share.none') }}</p>

        <ul v-else class="max-h-52 divide-y divide-foreground/5 overflow-auto">
          <li v-for="session in sessions" :key="session.id" class="flex items-center gap-2 py-2">
            <Lucide icon="Link" class="size-4 shrink-0 text-foreground/40" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm">{{ session.label || $t('documents.share.unlabelled') }}</p>
              <p class="truncate text-[11px] text-foreground/40">
                {{ $t('documents.share.expiresAt', { at: session.expires_at }) }} · {{ session.creator }}
              </p>
            </div>
            <button
              type="button"
              class="shrink-0 rounded p-1 text-foreground/40 hover:bg-danger/10 hover:text-danger"
              :disabled="submitting"
              :aria-label="$t('documents.share.revoke')"
              :title="$t('documents.share.revoke')"
              @click="revoke(session)"
            >
              <Lucide icon="Trash2" class="size-3.5" />
            </button>
          </li>
        </ul>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
