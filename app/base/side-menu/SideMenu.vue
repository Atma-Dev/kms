<script setup lang="ts">
import { Lucide, type Icon } from '~/base/ui/lucide'
import { slideUp, slideDown } from '~/utils/helper'
import { filterMenuByPermission } from '~/utils/menu'
import { cn } from '~/utils/cn'
import type { Permission } from '~/enums/Permission'
import _ from 'lodash'

interface Menu {
  icon?: Icon
  title?: string
  route_name?: string
  params?: any
  badge?: number
  sub_menu?: Menu[]
  permission?: Permission
}

const props = defineProps<{
  menu: (string | Menu)[]
}>()

const authStore = useAuthStore()
const { hasPermission } = authStore
const route = useRoute()
const router = useRouter()
const formattedMenu = ref<(string | Menu)[]>([])

// State to track open menus based on unique index/path
const openMenus = ref<Record<string, boolean>>({})

const pathMatches = (routeName: string): boolean =>
  routeName === '/'
    ? route.path === '/'
    : route.path === `/${routeName}` || route.path.startsWith(`/${routeName}/`)

// Longest matching route_name wins, so /a/b doesn't also light up /a
const bestMatch = computed(() => {
  const names: string[] = []
  const walk = (list: (string | Menu)[]) =>
    list.forEach((m) => {
      if (typeof m === 'string') return
      if (m.route_name) names.push(m.route_name)
      if (m.sub_menu) walk(m.sub_menu)
    })
  walk(props.menu)
  return names.filter(pathMatches).sort((a, b) => b.length - a.length)[0]
})

const isMenuActive = (menu: string | Menu): boolean => {
  if (typeof menu === 'string') return false
  if (menu.route_name) {
    return (
      menu.route_name === bestMatch.value ||
      route.name === menu.route_name ||
      _.kebabCase((route.name as string) || '') === menu.route_name
    )
  }
  if (menu.sub_menu) return menu.sub_menu.some(isMenuActive)
  return false
}

const findActiveSubMenus = (subMenu: Menu[]): boolean => subMenu.some(isMenuActive)

const initMenu = (menuList: (string | Menu)[], parentKey = 'menu') => {
  const newOpenMenus: Record<string, boolean> = {}

  const traverse = (list: (string | Menu)[], pk: string) => {
    list.forEach((item, index) => {
      const key = `${pk}-${index}`
      if (typeof item !== 'string' && item.sub_menu) {
        if (findActiveSubMenus(item.sub_menu)) {
          newOpenMenus[key] = true
        }
        traverse(item.sub_menu, key)
      }
    })
  }

  traverse(menuList, parentKey)
  openMenus.value = { ...openMenus.value, ...newOpenMenus }
}

// authStore.user is in the list so a permissions.changed push re-filters the
// menu immediately, instead of waiting for the next navigation.
watch(
  [() => props.menu, () => route.name, () => route.path, () => authStore.user],
  () => {
    formattedMenu.value = filterMenuByPermission(props.menu, hasPermission)
    initMenu(formattedMenu.value)
  },
  { immediate: true },
)

const toggleMenu = (key: string, event: MouseEvent) => {
  event.preventDefault()

  const el = (event.currentTarget as HTMLElement).nextElementSibling as HTMLElement
  if (el) {
    if (openMenus.value[key]) {
      slideUp(el, 300)
      openMenus.value[key] = false
    } else {
      slideDown(el, 300)
      openMenus.value[key] = true
    }
  }
}

const linkTo = async (menu: Menu, event: MouseEvent) => {
  event.preventDefault()
  if (menu.route_name) {
    // Use path-based navigation for absolute reliability in Nuxt
    const path = menu.route_name === 'dashboard-overview-1' ? '/' : `/${menu.route_name}`
    await navigateTo(path)
  }
}
</script>

<template>
  <ul class="scrollable">
    <template v-for="(menu, menuKey) in formattedMenu" :key="menuKey">
      <!-- BEGIN: First Child -->
      <li v-if="typeof menu === 'string'" class="side-menu__group-label">
        {{ $t(menu) }}
      </li>
      <li v-else>
        <!-- Toggle Sub-menu (Level 1) -->
        <a
          v-if="menu.sub_menu"
          href="javascript:;"
          @click.prevent="toggleMenu(`menu-${menuKey}`, $event)"
          :class="
            cn(
              'side-menu__link',
              isMenuActive(menu) ? 'side-menu__link--active' : '',
            )
          "
        >
          <Lucide
            v-if="menu.icon"
            class="side-menu__link__icon"
            :icon="menu.icon"
          />
          <div class="side-menu__link__title">{{ menu.title ? $t(menu.title) : '' }}</div>
          <div v-if="menu.badge" class="side-menu__link__badge">
            {{ menu.badge }}
          </div>
          <Lucide
            :class="
              cn(
                'side-menu__link__chevron transition-transform duration-300',
                openMenus[`menu-${menuKey}`] ? 'rotate-180' : '',
              )
            "
            icon="ChevronDown"
          />
        </a>
        <!-- Terminal Link (Level 1) -->
        <NuxtLink
          v-else
          :to="menu.route_name === 'dashboard-overview-1' ? '/' : `/${menu.route_name}`"
          :class="
            cn(
              'side-menu__link',
              isMenuActive(menu) ? 'side-menu__link--active' : '',
            )
          "
        >
          <Lucide
            v-if="menu.icon"
            class="side-menu__link__icon"
            :icon="menu.icon"
          />
          <div class="side-menu__link__title">{{ menu.title ? $t(menu.title) : '' }}</div>
          <div v-if="menu.badge" class="side-menu__link__badge">
            {{ menu.badge }}
          </div>
        </NuxtLink>

        <!-- BEGIN: Second Child -->
        <ul
          v-if="menu.sub_menu"
          :class="cn('hidden', { block: openMenus[`menu-${menuKey}`] })"
        >
          <li v-for="(subMenu, subMenuKey) in menu.sub_menu" :key="subMenuKey">
            <!-- Toggle Sub-menu (Level 2) -->
            <a
              v-if="subMenu.sub_menu"
              href="javascript:;"
              @click.prevent="toggleMenu(`menu-${menuKey}-${subMenuKey}`, $event)"
              :class="
                cn(
                  'side-menu__link',
                  isMenuActive(subMenu) ? 'side-menu__link--active' : '',
                )
              "
            >
              <Lucide
                v-if="subMenu.icon"
                class="side-menu__link__icon"
                :icon="subMenu.icon"
              />
              <div class="side-menu__link__title">
                {{ subMenu.title ? $t(subMenu.title) : '' }}
              </div>
              <div v-if="subMenu.badge" class="side-menu__link__badge">
                {{ subMenu.badge }}
              </div>
              <Lucide
                :class="
                  cn(
                    'side-menu__link__chevron transition-transform duration-300',
                    openMenus[`menu-${menuKey}-${subMenuKey}`] ? 'rotate-180' : '',
                  )
                "
                icon="ChevronDown"
              />
            </a>
            <!-- Terminal Link (Level 2) -->
            <NuxtLink
              v-else
              :to="`/${subMenu.route_name}`"
              :class="
                cn(
                  'side-menu__link',
                  isMenuActive(subMenu) ? 'side-menu__link--active' : '',
                )
              "
            >
              <Lucide
                v-if="subMenu.icon"
                class="side-menu__link__icon"
                :icon="subMenu.icon"
              />
              <div class="side-menu__link__title">
                {{ subMenu.title ? $t(subMenu.title) : '' }}
              </div>
              <div v-if="subMenu.badge" class="side-menu__link__badge">
                {{ subMenu.badge }}
              </div>
            </NuxtLink>

            <!-- BEGIN: Third Child -->
            <ul
              v-if="subMenu.sub_menu"
              :class="
                cn('hidden', { block: openMenus[`menu-${menuKey}-${subMenuKey}`] })
              "
            >
              <li
                v-for="(lastSubMenu, lastSubMenuKey) in subMenu.sub_menu"
                :key="lastSubMenuKey"
              >
                <!-- Terminal Link (Level 3) -->
                <NuxtLink
                  :to="`/${lastSubMenu.route_name}`"
                  :class="
                    cn(
                      'side-menu__link',
                      isMenuActive(lastSubMenu) ? 'side-menu__link--active' : '',
                    )
                  "
                >
                  <div class="side-menu__link__icon">
                    <Lucide v-if="lastSubMenu.icon" :icon="lastSubMenu.icon" />
                  </div>
                  <div class="side-menu__link__title">
                    {{ lastSubMenu.title ? $t(lastSubMenu.title) : '' }}
                  </div>
                  <div v-if="lastSubMenu.badge" class="side-menu__link__badge">
                    {{ lastSubMenu.badge }}
                  </div>
                </NuxtLink>
              </li>
            </ul>
            <!-- END: Third Child -->
          </li>
        </ul>
        <!-- END: Second Child -->
      </li>
      <!-- END: First Child -->
    </template>
  </ul>
</template>


