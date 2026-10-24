<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Badge } from '~/base/ui/badge'
import type Audit from '~/types/entities/audit'

const isShowModal = ref(false)
const audit = ref<Audit | null>(null)

const handleModal = (isShow: boolean, payload?: Audit) => {
  isShowModal.value = isShow
  if (payload) audit.value = payload
}

// before/after/tag/additional arrive as already-parsed JSON (or null) from
// the API — pretty-printed here the same way ModalActivityLogDetail formats
// its request/response bodies, so a trail with no field change (null) just
// renders as "— empty —" instead of the literal word "null".
const pretty = (value: unknown) => {
  if (value === null || value === undefined) return null
  return JSON.stringify(value, null, 2)
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(d: { open: boolean }) => { if (!d.open) handleModal(false) }"
  >
    <DialogContent class="sm:max-w-3xl">
      <DialogTitle>
        {{ audit?.action }}
      </DialogTitle>

      <div v-if="audit" class="flex flex-col gap-4 mt-4 max-h-[75vh] overflow-auto">
        <p v-if="audit.message" class="text-sm text-foreground/70">{{ audit.message }}</p>

        <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50">{{ $t('audits.detailModal.actorLabel') }}</p>
            <p>{{ audit.actor_name || '—' }} <span v-if="audit.actor_email" class="text-foreground/40">({{ audit.actor_email }})</span></p>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50">{{ $t('audits.detailModal.entryObjectLabel') }}</p>
            <p>{{ audit.entry_object_table }} #{{ audit.entry_object_id }}</p>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50">{{ $t('audits.detailModal.endpointLabel') }}</p>
            <p>
              <Badge v-if="audit.method" variant="secondary" look="outline" class="text-[10px] mr-1">{{ audit.method }}</Badge>
              {{ audit.endpoint || '—' }}
            </p>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50">{{ $t('common.createdAt') }}</p>
            <p>{{ audit.created_at }}</p>
          </div>
        </div>

        <div v-if="audit.trails.length === 0" class="text-sm text-foreground/40">
          {{ $t('audits.detailModal.noTrails') }}
        </div>

        <div
          v-for="trail in audit.trails"
          :key="trail.id"
          class="rounded-lg border border-foreground/10 p-3"
        >
          <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50 mb-2">
            {{ trail.object_table }} <span v-if="trail.object_id">#{{ trail.object_id }}</span>
          </p>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <p class="text-xs font-semibold text-foreground/50 mb-1">{{ $t('audits.detailModal.beforeLabel') }}</p>
              <pre
                v-if="pretty(trail.before)"
                class="rounded-lg border border-foreground/10 bg-foreground/5 p-3 text-xs whitespace-pre-wrap break-all"
              >{{ pretty(trail.before) }}</pre>
              <p v-else class="text-sm text-foreground/40">{{ $t('audits.detailModal.empty') }}</p>
            </div>
            <div>
              <p class="text-xs font-semibold text-foreground/50 mb-1">{{ $t('audits.detailModal.afterLabel') }}</p>
              <pre
                v-if="pretty(trail.after)"
                class="rounded-lg border border-foreground/10 bg-foreground/5 p-3 text-xs whitespace-pre-wrap break-all"
              >{{ pretty(trail.after) }}</pre>
              <p v-else class="text-sm text-foreground/40">{{ $t('audits.detailModal.empty') }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="flex justify-end mt-7">
        <Button variant="primary" @click="handleModal(false)">{{ $t('common.close') }}</Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
