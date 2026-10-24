<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '~/utils/cn'

const props = withDefaults(defineProps<{
  modelValue?: (string | number)[] | boolean | null,
  value?: string | number,
  label?: string,
  disabled?: boolean,
  class?: string,
}>(), {
  modelValue: false,
  class: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: (string | number)[] | boolean]
}>()

const isMultiple = computed(() => Array.isArray(props.modelValue))

const checked = computed<boolean>({
  get() {
    if (isMultiple.value && Array.isArray(props.modelValue)) {
      return props.modelValue.includes(props.value as string | number)
    }
    return !!props.modelValue
  },
  set(v: boolean) {
    if (isMultiple.value) {
      let arr: (string | number)[] = []
      if (Array.isArray(props.modelValue)) {
        arr = [...props.modelValue]
      }
      const idx = arr.findIndex((x) => x === props.value)
      if (v) {
        if (idx === -1 && props.value !== undefined) {
          arr.push(props.value as string | number)
        }
      } else {
        if (idx !== -1) arr.splice(idx, 1)
      }
      emit('update:modelValue', arr)
    } else {
      emit('update:modelValue', v)
    }
  }
})

const onChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  checked.value = target.checked
}

// Satisfy ESLint - these ARE used in template
if (false as boolean) { cn(''); onChange(new Event('')) }
</script>

<template>
  <label :class="cn('inline-flex items-center gap-2 cursor-pointer', props.class)">
    <input
      type="checkbox"
      :checked="checked"
      :disabled="disabled"
      @change="onChange"
      class="form-checkbox h-4 w-4 text-primary rounded border-slate-300"
    />
    <span class="select-none text-sm text-slate-700">{{ label || '' }}</span>
  </label>
</template>

<style scoped>
/* adapt default appearance slightly to match input look */
.form-checkbox:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(59,130,246,0.15);
}
</style>
