<script lang="ts" setup>
import { Lucide } from '~/base/ui/lucide'
import RichTextEditor from '~/components/features/documents/RichTextEditor.vue'
import DocumentProtection from '~/components/features/documents/DocumentProtection.vue'
import DocumentPages from '~/components/features/documents/DocumentPages.vue'
import { useProtectionService } from '~/services/ProtectionService'
import type { SharedDocument } from '~/types/entities/protection'

/**
 * A document opened by share link (FR-5), by somebody with no account.
 *
 * `public` so the auth middleware lets it through, `layout: false` because the
 * app shell around it — the nav, the search, the space list — is exactly what
 * a link recipient is not being given. They get one document.
 *
 * Everything on the page comes from the token: which document, whose name the
 * watermark carries, and when the link dies. Nothing here can widen that.
 */
definePageMeta({
  layout: false,
  public: true,
})

const route = useRoute()
const token = computed(() => route.params.token as string)

const { openShared, loading } = useProtectionService()
const shared = ref<SharedDocument | null>(null)

onMounted(async () => {
  shared.value = await openShared(token.value)
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <div v-if="loading" class="py-24 text-center text-sm text-foreground/40">
      {{ $t('common.loading') }}
    </div>

    <!-- One message for every failure: expired, revoked, mistyped, or a
         document taken down since the link went out. Telling them apart would
         tell whoever is holding a wrong token which part they got right. -->
    <div v-else-if="!shared" class="mx-auto max-w-md px-6 py-24 text-center">
      <Lucide icon="LinkOff" class="mx-auto size-10 text-foreground/25" />
      <h1 class="mt-4 text-lg font-medium">{{ $t('documents.shared.unavailableTitle') }}</h1>
      <p class="mt-2 text-sm text-foreground/50">{{ $t('documents.shared.unavailableBody') }}</p>
    </div>

    <div v-else class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <header class="mb-6 border-b border-foreground/10 pb-4">
        <h1 class="text-xl font-medium">{{ shared.title }}</h1>
        <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-foreground/45">
          <span class="inline-flex items-center gap-1">
            <Lucide icon="Clock" class="size-3.5" />
            {{ $t('documents.shared.expiresAt', { at: shared.expires_at }) }}
          </span>
          <span>·</span>
          <!-- Says whose name is on this copy. The recipient has no account,
               so the trace runs to whoever shared it — and saying so out loud
               is most of what makes a watermark work. -->
          <span class="inline-flex items-center gap-1">
            <Lucide icon="ShieldCheck" class="size-3.5" />
            {{ shared.watermark }}
          </span>
        </p>
      </header>

      <DocumentPages
        v-if="shared.protection_level === 'strict'"
        :document-id="token"
        :pages="shared.pages ?? []"
      />

      <DocumentProtection v-else :text="shared.watermark" protect>
        <RichTextEditor :model-value="shared.content" readonly />
      </DocumentProtection>
    </div>
  </div>
</template>
