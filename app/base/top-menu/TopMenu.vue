<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Lucide, type Icon } from '~/base/ui/lucide'
import { slideUp, slideDown } from '~/utils/helper'
import { filterMenuByPermission } from '~/utils/menu'
import type { Permission } from '~/enums/Permission'

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

const { hasPermission } = useAuthStore()
const route = useRoute()
const router = useRouter()
const formattedMenu = ref<(string | Menu)[]>([])

// State to track open menus based on unique index/path
const openMenus = ref<Record<string, boolean>>({})

watch(
  () => props.menu,
  () => {
    formattedMenu.value = filterMenuByPermission(props.menu, hasPermission)
  },
  { immediate: true },
)

const pathMatches = (routeName: string): boolean =>
  route.path === `/${routeName}` || route.path.startsWith(`/${routeName}/`)

// Longest matching route_name wins, so /deposits/request doesn't also light
// up /deposits — same fix as SideMenu.vue's bestMatch.
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
  if (menu.route_name && menu.route_name === bestMatch.value) return true
  if (menu.sub_menu) {
    return menu.sub_menu.some((sub) => isMenuActive(sub))
  }
  return false
}

const linkTo = (menu: Menu, event: MouseEvent) => {
  event.preventDefault()
  if (menu.route_name) {
    // route_name is a path fragment (e.g. "deposits/request"), not a vue-router
    // route name — nested pages get hyphenated names ("deposits-request"), so
    // pushing { name: menu.route_name } fails to resolve. Path-based push
    // matches SideMenu.vue's navigateTo(`/${route_name}`) approach.
    router.push(`/${menu.route_name}`)
  }
}

const toggleMenu = (key: string, event: MouseEvent) => {
  event.preventDefault()

  // Get the <ul> element that is a sibling of the clicked <a>
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
</script>

<template>
  <ul class="top-menu">
    <template v-for="(menu, menuKey) in formattedMenu" :key="menuKey">
      <li v-if="typeof menu !== 'string'">
        <a
          href=""
          @click="(event) => {
            if (menu.sub_menu) {
              toggleMenu(`menu-${menuKey}`, event)
            } else {
              linkTo(menu, event)
            }
          }"
          :class="[
            'top-menu__link',
            isMenuActive(menu) ? 'top-menu__link--active' : '',
          ]"
        >
          <div class="top-menu__link__icon">
            <Lucide v-if="menu.icon" :icon="menu.icon" />
          </div>
          <div class="top-menu__link__title">
            {{ menu.title }}
            <Lucide
              v-if="menu.sub_menu"
              :class="[
                'top-menu__link__chevron transition-transform duration-300',
                openMenus[`menu-${menuKey}`] ? 'rotate-180' : '',
              ]"
              icon="ChevronDown"
            />
          </div>
        </a>
        <!-- BEGIN: Second Child -->
        <ul v-if="menu.sub_menu" class="hidden">
          <li v-for="(subMenu, subMenuKey) in menu.sub_menu" :key="subMenuKey">
            <a
              href=""
              @click="(event) => {
                if (subMenu.sub_menu) {
                  toggleMenu(`menu-${menuKey}-${subMenuKey}`, event)
                } else {
                  linkTo(subMenu, event)
                }
              }"
              :class="[
                'top-menu__link',
                isMenuActive(subMenu) ? 'top-menu__link--active' : '',
              ]"
            >
              <div class="top-menu__link__icon">
                <Lucide v-if="subMenu.icon" :icon="subMenu.icon" />
              </div>
              <div class="top-menu__link__title">
                {{ subMenu.title }}
                <Lucide
                  v-if="subMenu.sub_menu"
                  :class="[
                    'top-menu__link__chevron transition-transform duration-300',
                    openMenus[`menu-${menuKey}-${subMenuKey}`] ? 'rotate-180' : '',
                  ]"
                  icon="ChevronDown"
                />
              </div>
            </a>
            <!-- BEGIN: Third Child -->
            <ul v-if="subMenu.sub_menu" class="hidden">
              <li
                v-for="(lastSubMenu, lastSubMenuKey) in subMenu.sub_menu"
                :key="lastSubMenuKey"
              >
                <a
                  href=""
                  @click.prevent="linkTo(lastSubMenu, $event)"
                  :class="[
                    'top-menu__link',
                    isMenuActive(lastSubMenu) ? 'top-menu__link--active' : '',
                  ]"
                >
                  <div class="top-menu__link__icon">
                    <Lucide v-if="lastSubMenu.icon" :icon="lastSubMenu.icon" />
                  </div>
                  <div class="top-menu__link__title">
                    {{ lastSubMenu.title }}
                  </div>
                </a>
              </li>
            </ul>
            <!-- END: Third Child -->
          </li>
        </ul>
        <!-- END: Second Child -->
      </li>
    </template>
  </ul>
</template>
