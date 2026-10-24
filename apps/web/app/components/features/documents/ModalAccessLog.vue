<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Input } from '~/base/ui/input'
import { Label } from '~/base/ui/label'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useProtectionService } from '~/services/ProtectionService'
import type { AccessAction } from '~/types/entities/protection'
import type { BadgeVariants } from '~/base/ui/styles/badge.styles'

/**
 * The access trail for one document (FR-4): who opened it, from where, and
 * what they did with it.
 *
 * Read-only by construction — the API has no route that edits or removes a
 * row, so there is nothing for this to offer. A dialog rather than a page: an
 * investigation starts from the document someone is worried about.
 */
const props = defineProps<{ documentId: string }>()

const { logs, meta, loading, getAccessLog } = useProtectionService()

const isShowModal = ref(false)
const page = ref(1)
const filters = reactive({ action: '', from: '', to: '' })

const { t } = useI18n()
const actionOptions = computed(() => [
  { id: '', name: t('documents.accessLog.allActions') },
  { id: 'view', name: t('documents.accessLog.action.view') },
  { id: 'download', name: t('documents.accessLog.action.download') },
  { id: 'print', name: t('documents.accessLog.action.print') },
  { id: 'export_pdf', name: t('documents.accessLog.action.export_pdf') },
])

type BadgeVariant = NonNullable<BadgeVariants['variant']>

const ACTION_VARIANT: Record<AccessAction, BadgeVariant> = {
  view: 'secondary',
  download: 'primary',
  print: 'pending',
  export_pdf: 'warning',
}

const load = () => getAccessLog(props.documentId, page.value, 15, filters)

const handleModal = async (isShow: boolean) => {
  isShowModal.value = isShow
  if (!isShow) return
  page.value = 1
  await load()
}

// Filters and paging both go through one loader rather than each calling the
// service: two callers is how a filter change ends up on page 4 of the old
// result set.
watch(filters, () => {
  page.value = 1
  load()
})

const goTo = (next: number) => {
  page.value = next
  load()
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => handleModal(d.open)">
    <DialogContent class="sm:max-w-3xl">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.accessLog.title') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <p class="mt-3 rounded-xl bg-foreground/5 px-3 py-2 text-xs text-foreground/60">
        {{ $t('documents.accessLog.hint') }}
      </p>

      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div>
          <Label>{{ $t('documents.accessLog.actionLabel') }}</Label>
          <div class="mt-2">
            <TomSelect v-model="filters.action" :options="actionOptions" :placeholder="$t('documents.accessLog.allActions')" />
          </div>
        </div>
        <div>
          <!-- No placeholder: browsers ignore it on a date input and render
               their own format hint. -->
          <Label for="log-from">{{ $t('documents.accessLog.fromLabel') }}</Label>
          <Input id="log-from" v-model="filters.from" type="date" class="mt-2" />
        </div>
        <div>
          <Label for="log-to">{{ $t('documents.accessLog.toLabel') }}</Label>
          <Input id="log-to" v-model="filters.to" type="date" class="mt-2" />
        </div>
      </div>

      <div class="mt-4 max-h-96 overflow-auto">
        <p v-if="loading" class="py-8 text-center text-sm text-foreground/40">{{ $t('common.loading') }}</p>
        <p v-else-if="!logs.length" class="py-8 text-center text-sm text-foreground/40">
          {{ $t('documents.accessLog.empty') }}
        </p>

        <ul v-else class="divide-y divide-foreground/5">
          <li v-for="entry in logs" :key="entry.id" class="flex items-start gap-3 py-2.5">
            <Badge :variant="ACTION_VARIANT[entry.action]" look="outline" class="mt-0.5 shrink-0 text-[10px]">
              {{ $t(`documents.accessLog.action.${entry.action}`) }}
            </Badge>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm">
                {{ entry.user || $t('documents.accessLog.sharedLinkReader') }}
                <span v-if="entry.user_email" class="text-foreground/40">· {{ entry.user_email }}</span>
              </p>
              <p class="truncate text-[11px] text-foreground/40">
                {{ entry.created_at }} · {{ entry.ip_address || '—' }}
              </p>
              <p v-if="entry.user_agent" class="truncate text-[11px] text-foreground/30">{{ entry.user_agent }}</p>
            </div>
          </li>
        </ul>
      </div>

      <div v-if="meta.last_page > 1" class="mt-3 flex items-center justify-between text-xs text-foreground/50">
        <span>{{ $t('documents.accessLog.page', { current: meta.current_page, last: meta.last_page }) }}</span>
        <div class="flex gap-1">
          <button
            type="button"
            class="rounded p-1.5 hover:bg-foreground/5 disabled:opacity-30"
            :disabled="meta.current_page <= 1 || loading"
            :aria-label="$t('common.previous')"
            @click="goTo(meta.current_page - 1)"
          >
            <Lucide icon="ChevronLeft" class="size-4" />
          </button>
          <button
            type="button"
            class="rounded p-1.5 hover:bg-foreground/5 disabled:opacity-30"
            :disabled="meta.current_page >= meta.last_page || loading"
            :aria-label="$t('common.next')"
            @click="goTo(meta.current_page + 1)"
          >
            <Lucide icon="ChevronRight" class="size-4" />
          </button>
        </div>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
