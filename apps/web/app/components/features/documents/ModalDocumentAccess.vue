<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import type { AccessLevel, AccessSubjectType, DocumentAccess } from '~/types/entities/attachment'
import { useDocumentService } from '~/services/DocumentService'
import { useLookupService } from '~/services/LookupService'

const props = defineProps<{ documentId: string }>()
const emit = defineEmits<{ (e: 'changed'): void }>()

const { getAccess, grantAccess, revokeAccess, submitting } = useDocumentService()
const users = useLookupService('users')
const roles = useLookupService('roles')

const isShowModal = ref(false)
const grants = ref<DocumentAccess[]>([])
const loading = ref(false)

const subjectType = ref<AccessSubjectType>('user')
const subjectId = ref('')
const permission = ref<AccessLevel>('view')

const { t } = useI18n()
const levelOptions = computed(() => [
  { id: 'view', name: t('documents.access.levelView') },
  { id: 'edit', name: t('documents.access.levelEdit') },
  { id: 'manage', name: t('documents.access.levelManage') },
])

const subjectOptions = computed(() =>
  subjectType.value === 'user' ? users.options.value : roles.options.value,
)

const nameFor = (grant: DocumentAccess) => {
  const list = grant.subject_type === 'user' ? users.options.value : roles.options.value
  return list.find((o) => o.id === grant.subject_id)?.name ?? grant.subject_id
}

const load = async () => {
  loading.value = true
  grants.value = await getAccess(props.documentId)
  loading.value = false
}

const handleModal = async (isShow: boolean) => {
  isShowModal.value = isShow
  if (!isShow) return
  subjectId.value = ''
  permission.value = 'view'
  await Promise.all([users.load(), roles.load(), load()])
}

const add = async () => {
  if (!subjectId.value) return
  if (await grantAccess(props.documentId, subjectType.value, subjectId.value, permission.value)) {
    subjectId.value = ''
    await load()
    emit('changed')
  }
}

const remove = async (grant: DocumentAccess) => {
  if (await revokeAccess(props.documentId, grant.id)) {
    await load()
    emit('changed')
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }">
    <DialogContent class="sm:max-w-lg">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.access.title') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <!-- Says plainly which way this works, because the opposite reading is
           the natural one: adding the first person locks everyone else out. -->
      <p class="mt-3 rounded-xl bg-foreground/5 px-3 py-2 text-xs text-foreground/60">
        {{ grants.length ? $t('documents.access.restrictedHint') : $t('documents.access.openHint') }}
      </p>

      <div class="mt-4 grid gap-3">
        <div class="flex gap-2">
          <Button
            v-for="type in (['user', 'role'] as AccessSubjectType[])"
            :key="type"
            size="sm"
            :look="subjectType === type ? 'filled' : 'outline'"
            :variant="subjectType === type ? 'primary' : 'ghost'"
            @click="subjectType = type; subjectId = ''"
          >
            {{ type === 'user' ? $t('documents.access.subjectUser') : $t('documents.access.subjectRole') }}
          </Button>
        </div>

        <div>
          <Label>{{ subjectType === 'user' ? $t('documents.access.pickUser') : $t('documents.access.pickRole') }}</Label>
          <TomSelect v-model="subjectId" class="mt-2" :options="subjectOptions" />
        </div>

        <div>
          <Label>{{ $t('documents.access.levelLabel') }}</Label>
          <TomSelect v-model="permission" class="mt-2" :options="levelOptions" />
        </div>

        <Button variant="primary" :disabled="submitting || !subjectId" @click="add">
          <Lucide icon="Plus" class="mr-1.5 size-4" />
          {{ $t('documents.access.grant') }}
        </Button>
      </div>

      <div class="mt-5">
        <p class="mb-2 text-sm font-semibold">{{ $t('documents.access.currentTitle') }}</p>
        <p v-if="loading" class="text-sm text-foreground/40">{{ $t('common.loading') }}</p>
        <p v-else-if="!grants.length" class="text-sm text-foreground/40">{{ $t('documents.access.none') }}</p>

        <ul v-else class="max-h-52 divide-y divide-foreground/5 overflow-auto">
          <li v-for="grant in grants" :key="grant.id" class="flex items-center gap-2 py-2">
            <Lucide :icon="grant.subject_type === 'user' ? 'User' : 'Shield'" class="size-4 shrink-0 text-foreground/40" />
            <span class="min-w-0 flex-1 truncate text-sm">{{ nameFor(grant) }}</span>
            <Badge variant="secondary" look="outline" class="shrink-0 text-[10px]">
              {{ $t(`documents.access.level${grant.permission.charAt(0).toUpperCase() + grant.permission.slice(1)}`) }}
            </Badge>
            <button
              type="button"
              class="shrink-0 rounded p-1 text-foreground/40 hover:bg-danger/10 hover:text-danger"
              :disabled="submitting"
              :aria-label="$t('documents.access.revoke')"
              :title="$t('documents.access.revoke')"
              @click="remove(grant)"
            >
              <Lucide icon="Trash2" class="size-3.5" />
            </button>
          </li>
        </ul>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
