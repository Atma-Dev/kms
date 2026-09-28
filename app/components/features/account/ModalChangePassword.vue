<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Label } from '~/base/ui/label'
import { Lucide } from '~/base/ui/lucide'
import { useProfileService } from '~/services/ProfileService'

const { t } = useI18n()
const { updatePassword, submitting } = useProfileService()

const isShowModal = ref(false)
const form = reactive({ current_password: '', password: '', password_confirmation: '' })
const error = ref('')

const handleModal = (isShow: boolean) => {
  isShowModal.value = isShow
  error.value = ''
  form.current_password = ''
  form.password = ''
  form.password_confirmation = ''
}

// Checked here as well as server-side: the API rejects a mismatched pair, but
// letting the round-trip answer it would clear the fields for a typo.
const onSubmit = async () => {
  error.value = ''
  if (form.password.length < 8) {
    error.value = t('auth.passwordTooShort')
    return
  }
  if (form.password !== form.password_confirmation) {
    error.value = t('auth.passwordMismatch')
    return
  }

  if (await updatePassword({ ...form })) handleModal(false)
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(details: { open: boolean }) => { isShowModal = details.open }"
  >
    <DialogContent class="sm:max-w-md">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('auth.changePassword') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <form class="mt-6 grid gap-4" @submit.prevent="onSubmit">
        <div>
          <Label for="current_password">{{ $t('auth.currentPassword') }} <RequiredMark /></Label>
          <Input
            id="current_password"
            v-model="form.current_password"
            class="mt-2"
            type="password"
            autocomplete="current-password"
            required
            :placeholder="$t('auth.currentPasswordPlaceholder')"
          />
          <FormFeedback feedbackKey="current_password" />
        </div>

        <div>
          <Label for="new_password">{{ $t('auth.newPassword') }} <RequiredMark /></Label>
          <Input
            id="new_password"
            v-model="form.password"
            class="mt-2"
            type="password"
            autocomplete="new-password"
            required
            :placeholder="$t('auth.newPasswordPlaceholder')"
          />
          <FormFeedback feedbackKey="password" />
        </div>

        <div>
          <Label for="confirm_password">{{ $t('auth.confirmPassword') }} <RequiredMark /></Label>
          <Input
            id="confirm_password"
            v-model="form.password_confirmation"
            class="mt-2"
            type="password"
            autocomplete="new-password"
            required
            :placeholder="$t('auth.confirmPasswordPlaceholder')"
          />
          <FormFeedback feedbackKey="password_confirmation" />
        </div>

        <p v-if="error" class="text-sm text-danger">{{ error }}</p>

        <div class="mt-2 flex justify-end gap-2">
          <Button type="button" @click="handleModal(false)">
            <Lucide icon="SquareX" />
            {{ $t('common.close') }}
          </Button>
          <Button variant="primary" type="submit" :disabled="submitting">
            <Lucide icon="Save" />
            {{ $t('common.save') }}
          </Button>
        </div>
      </form>
    </DialogContent>
  </DialogRoot>
</template>
