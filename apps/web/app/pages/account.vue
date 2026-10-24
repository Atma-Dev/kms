<script setup lang="ts">
import type { Icon } from '~/base/ui/lucide'
import SectionNav from '~/components/SectionNav.vue'

// Adding an account resource = one entry here + one page file.
const menus: { title: string; description: string; to: string; icon: Icon }[] = [
  { title: 'account.nav.usersTitle', description: 'account.nav.usersDescription', to: '/account/users', icon: 'UsersRound' },
  { title: 'account.nav.rolesTitle', description: 'account.nav.rolesDescription', to: '/account/roles', icon: 'UserLock' },
  {
    title: 'account.nav.permissionsTitle',
    description: 'account.nav.permissionsDescription',
    to: '/account/permissions',
    icon: 'Shield',
  },
  { title: 'account.nav.cacheTitle', description: 'account.nav.cacheDescription', to: '/account/cache', icon: 'DatabaseZap' },
]
</script>

<template>
  <SectionNav :menus="menus" />
</template>
