<script setup lang="ts">
import {Database} from 'lucide-vue-next';
import {defineProps} from 'vue';
import { Box } from '~/base/ui/box'

const props = defineProps({
    title: {
        type: String,
        default: undefined
    },
    message: {
        type: String,
        default: undefined
    },
    isWithBox :{
        required: false,
        type: Boolean,
        default: false
    }
});

const { t } = useI18n()
</script>

<template>
    <component :is="isWithBox ? Box : 'div'">
        <div class="flex flex-col items-center justify-center text-center p-8 text-gray-500">
            <Database class="mb-4 text-gray-700 text-2xl"/>
            <h3 class="text-xl font-semibold mb-2">{{ props.title ?? t('common.emptyDataTitle') }}</h3>
            <p>{{ props.message ?? t('common.emptyDataMessage') }}</p>
        </div>
    </component>
</template>
